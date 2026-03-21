package dynamodb

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// TableStore manages all DynamoDB tables in memory.
type TableStore struct {
	mu        sync.RWMutex
	tables    map[string]*Table
	accountID string
	region    string
}

// NewTableStore creates an empty TableStore.
func NewTableStore(accountID, region string) *TableStore {
	return &TableStore{
		tables:    make(map[string]*Table),
		accountID: accountID,
		region:    region,
	}
}

func (s *TableStore) tableARN(name string) string {
	return fmt.Sprintf("arn:aws:dynamodb:%s:%s:table/%s", s.region, s.accountID, name)
}

// CreateTable creates a new table. Returns ResourceInUseException if it already exists.
func (s *TableStore) CreateTable(name string, keySchema []KeySchemaElement, attrDefs []AttributeDefinition, billingMode string, pt *ProvisionedThroughput, gsis []GSI, lsis []LSI) (*Table, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tables[name]; ok {
		return nil, service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Table already exists: %s", name), http.StatusBadRequest)
	}

	if billingMode == "" {
		billingMode = "PROVISIONED"
	}

	gsiItems := make(map[string][]Item)
	for _, gsi := range gsis {
		gsiItems[gsi.IndexName] = nil
	}
	lsiItems := make(map[string][]Item)
	for _, lsi := range lsis {
		lsiItems[lsi.IndexName] = nil
	}

	table := &Table{
		Name:                  name,
		KeySchema:             keySchema,
		AttributeDefinitions:  attrDefs,
		Items:                 nil,
		Status:                "ACTIVE",
		CreationDateTime:      float64(time.Now().Unix()),
		ItemCount:             0,
		BillingMode:           billingMode,
		ProvisionedThroughput: pt,
		GSIs:                  gsis,
		LSIs:                  lsis,
		GSIItems:              gsiItems,
		LSIItems:              lsiItems,
	}
	s.tables[name] = table
	return table, nil
}

// DeleteTable removes a table. Returns ResourceNotFoundException if not found.
func (s *TableStore) DeleteTable(name string) (*Table, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	table, ok := s.tables[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Requested resource not found: Table: %s not found", name), http.StatusBadRequest)
	}

	table.Status = "DELETING"
	delete(s.tables, name)
	return table, nil
}

// DescribeTable returns table metadata. Returns ResourceNotFoundException if not found.
func (s *TableStore) DescribeTable(name string) (*Table, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table, ok := s.tables[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Requested resource not found: Table: %s not found", name), http.StatusBadRequest)
	}
	return table, nil
}

// ListTables returns the names of all tables.
func (s *TableStore) ListTables() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.tables))
	for name := range s.tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// getTable returns a table by name (caller must hold at least read lock).
func (s *TableStore) getTable(name string) (*Table, *service.AWSError) {
	table, ok := s.tables[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Requested resource not found: Table: %s not found", name), http.StatusBadRequest)
	}
	return table, nil
}

// PutItem adds or replaces an item in the specified table.
func (s *TableStore) PutItem(tableName string, item Item) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putItemLocked(tableName, item)
}

// putItemLocked adds or replaces an item (caller must hold write lock).
func (s *TableStore) putItemLocked(tableName string, item Item) *service.AWSError {
	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	// Replace existing item with same key, or append.
	for i, existing := range table.Items {
		if table.keyMatchesItem(item, existing) {
			table.deindexItem(existing)
			table.Items[i] = item
			table.indexItem(item)
			return nil
		}
	}
	table.Items = append(table.Items, item)
	table.ItemCount = int64(len(table.Items))
	table.indexItem(item)
	return nil
}

// GetItem retrieves an item by key from the specified table.
func (s *TableStore) GetItem(tableName string, key Item, projExpr string, exprNames map[string]string) (Item, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	for _, item := range table.Items {
		if table.keyMatchesItem(key, item) {
			result := copyItem(item)
			if projExpr != "" {
				result = applyProjection(result, projExpr, exprNames)
			}
			return result, nil
		}
	}
	return nil, nil // not found returns nil item, not an error
}

// DeleteItem removes an item by key from the specified table.
func (s *TableStore) DeleteItem(tableName string, key Item) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteItemLocked(tableName, key)
}

// deleteItemLocked removes an item by key (caller must hold write lock).
func (s *TableStore) deleteItemLocked(tableName string, key Item) *service.AWSError {
	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	for i, item := range table.Items {
		if table.keyMatchesItem(key, item) {
			table.deindexItem(item)
			table.Items = append(table.Items[:i], table.Items[i+1:]...)
			table.ItemCount = int64(len(table.Items))
			return nil
		}
	}
	return nil // deleting non-existent item is not an error
}

// UpdateItem updates an item using an UpdateExpression. Creates the item if it doesn't exist.
func (s *TableStore) UpdateItem(tableName string, key Item, updateExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, returnValues string) (Item, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateItemLocked(tableName, key, updateExpr, exprNames, exprValues, returnValues)
}

// updateItemLocked updates an item (caller must hold write lock).
func (s *TableStore) updateItemLocked(tableName string, key Item, updateExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, returnValues string) (Item, *service.AWSError) {
	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	// Find or create item.
	var target Item
	var idx int = -1
	for i, item := range table.Items {
		if table.keyMatchesItem(key, item) {
			target = copyItem(item)
			idx = i
			break
		}
	}
	if target == nil {
		target = copyItem(key)
	}

	target = parseUpdateExpression(target, updateExpr, exprNames, exprValues)

	if idx >= 0 {
		table.deindexItem(table.Items[idx])
		table.Items[idx] = target
	} else {
		table.Items = append(table.Items, target)
		table.ItemCount = int64(len(table.Items))
	}
	table.indexItem(target)

	switch returnValues {
	case "ALL_NEW":
		return copyItem(target), nil
	case "NONE", "":
		return nil, nil
	default:
		return nil, nil
	}
}

// Query finds items matching a key condition expression, applies filter and projection.
func (s *TableStore) Query(tableName string, indexName string, keyCondExpr string, filterExpr string, projExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, scanForward *bool, limit int) ([]Item, int, int, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return nil, 0, 0, awsErr
	}

	// Determine source items and range key based on index.
	var sourceItems []Item
	var rk string
	if indexName != "" {
		// Check GSIs first, then LSIs.
		found := false
		for _, gsi := range table.GSIs {
			if gsi.IndexName == indexName {
				sourceItems = table.GSIItems[indexName]
				rk = gsiRangeKeyName(gsi.KeySchema)
				found = true
				break
			}
		}
		if !found {
			for _, lsi := range table.LSIs {
				if lsi.IndexName == indexName {
					sourceItems = table.LSIItems[indexName]
					rk = gsiRangeKeyName(lsi.KeySchema)
					found = true
					break
				}
			}
		}
		if !found {
			return nil, 0, 0, service.NewAWSError("ValidationException",
				fmt.Sprintf("The table does not have the specified index: %s", indexName), http.StatusBadRequest)
		}
	} else {
		sourceItems = table.Items
		rk = table.rangeKeyName()
	}

	// Find items matching key condition.
	var matched []Item
	for _, item := range sourceItems {
		if evaluateCondition(keyCondExpr, item, exprNames, exprValues) {
			matched = append(matched, item)
		}
	}

	scannedCount := len(matched)

	// Sort by sort key if present.
	if rk != "" {
		forward := true
		if scanForward != nil {
			forward = *scanForward
		}
		sort.SliceStable(matched, func(i, j int) bool {
			cmp, ok := compareValues(matched[i][rk], matched[j][rk])
			if !ok {
				return false
			}
			if forward {
				return cmp < 0
			}
			return cmp > 0
		})
	}

	// Apply filter expression.
	var filtered []Item
	if filterExpr != "" {
		for _, item := range matched {
			if evaluateCondition(filterExpr, item, exprNames, exprValues) {
				filtered = append(filtered, item)
			}
		}
	} else {
		filtered = matched
	}

	// Apply limit.
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// Apply projection.
	results := make([]Item, len(filtered))
	for i, item := range filtered {
		results[i] = copyItem(item)
		if projExpr != "" {
			results[i] = applyProjection(results[i], projExpr, exprNames)
		}
	}

	return results, len(results), scannedCount, nil
}

// Scan iterates all items, applies filter and projection.
func (s *TableStore) Scan(tableName string, filterExpr string, projExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, limit int) ([]Item, int, int, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return nil, 0, 0, awsErr
	}

	scannedCount := len(table.Items)

	var filtered []Item
	for _, item := range table.Items {
		if filterExpr != "" {
			if !evaluateCondition(filterExpr, item, exprNames, exprValues) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	results := make([]Item, len(filtered))
	for i, item := range filtered {
		results[i] = copyItem(item)
		if projExpr != "" {
			results[i] = applyProjection(results[i], projExpr, exprNames)
		}
	}

	return results, len(results), scannedCount, nil
}

// TransactWriteItems executes a transactional write across multiple tables.
func (s *TableStore) TransactWriteItems(items []transactWriteItem) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Phase 1: Validate all conditions.
	reasons := make([]cancellationReason, len(items))
	anyFailed := false

	for i, txItem := range items {
		reasons[i] = cancellationReason{Code: "None"}

		if txItem.ConditionCheck != nil {
			cc := txItem.ConditionCheck
			table, awsErr := s.getTable(cc.TableName)
			if awsErr != nil {
				reasons[i] = cancellationReason{Code: "ValidationException", Message: awsErr.Message}
				anyFailed = true
				continue
			}
			var found Item
			for _, item := range table.Items {
				if table.keyMatchesItem(cc.Key, item) {
					found = item
					break
				}
			}
			if found == nil || !evaluateCondition(cc.ConditionExpression, found, cc.ExpressionAttributeNames, cc.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
				anyFailed = true
			}
		}

		if txItem.Put != nil && txItem.Put.ConditionExpression != "" {
			p := txItem.Put
			table, awsErr := s.getTable(p.TableName)
			if awsErr != nil {
				reasons[i] = cancellationReason{Code: "ValidationException", Message: awsErr.Message}
				anyFailed = true
				continue
			}
			var found Item
			for _, item := range table.Items {
				if table.keyMatchesItem(p.Item, item) {
					found = item
					break
				}
			}
			// For Put with condition, evaluate against existing item (or empty if not found).
			if found == nil {
				found = make(Item)
			}
			if !evaluateCondition(p.ConditionExpression, found, p.ExpressionAttributeNames, p.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
				anyFailed = true
			}
		}

		if txItem.Delete != nil && txItem.Delete.ConditionExpression != "" {
			d := txItem.Delete
			table, awsErr := s.getTable(d.TableName)
			if awsErr != nil {
				reasons[i] = cancellationReason{Code: "ValidationException", Message: awsErr.Message}
				anyFailed = true
				continue
			}
			var found Item
			for _, item := range table.Items {
				if table.keyMatchesItem(d.Key, item) {
					found = item
					break
				}
			}
			if found == nil {
				found = make(Item)
			}
			if !evaluateCondition(d.ConditionExpression, found, d.ExpressionAttributeNames, d.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
				anyFailed = true
			}
		}

		if txItem.Update != nil && txItem.Update.ConditionExpression != "" {
			u := txItem.Update
			table, awsErr := s.getTable(u.TableName)
			if awsErr != nil {
				reasons[i] = cancellationReason{Code: "ValidationException", Message: awsErr.Message}
				anyFailed = true
				continue
			}
			var found Item
			for _, item := range table.Items {
				if table.keyMatchesItem(u.Key, item) {
					found = item
					break
				}
			}
			if found == nil {
				found = make(Item)
			}
			if !evaluateCondition(u.ConditionExpression, found, u.ExpressionAttributeNames, u.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
				anyFailed = true
			}
		}

		// Also validate that tables exist for non-condition operations.
		if txItem.Put != nil && txItem.Put.ConditionExpression == "" {
			if _, awsErr := s.getTable(txItem.Put.TableName); awsErr != nil {
				reasons[i] = cancellationReason{Code: "ValidationException", Message: awsErr.Message}
				anyFailed = true
			}
		}
		if txItem.Delete != nil && txItem.Delete.ConditionExpression == "" {
			if _, awsErr := s.getTable(txItem.Delete.TableName); awsErr != nil {
				reasons[i] = cancellationReason{Code: "ValidationException", Message: awsErr.Message}
				anyFailed = true
			}
		}
		if txItem.Update != nil && txItem.Update.ConditionExpression == "" {
			if _, awsErr := s.getTable(txItem.Update.TableName); awsErr != nil {
				reasons[i] = cancellationReason{Code: "ValidationException", Message: awsErr.Message}
				anyFailed = true
			}
		}
	}

	if anyFailed {
		return service.NewAWSError("TransactionCanceledException",
			"Transaction cancelled, please refer cancellation reasons for specific reasons ["+formatReasons(reasons)+"]",
			http.StatusBadRequest)
	}

	// Phase 2: Execute all writes.
	for _, txItem := range items {
		if txItem.Put != nil {
			s.putItemLocked(txItem.Put.TableName, txItem.Put.Item)
		}
		if txItem.Delete != nil {
			s.deleteItemLocked(txItem.Delete.TableName, txItem.Delete.Key)
		}
		if txItem.Update != nil {
			u := txItem.Update
			s.updateItemLocked(u.TableName, u.Key, u.UpdateExpression, u.ExpressionAttributeNames, u.ExpressionAttributeValues, "NONE")
		}
	}

	return nil
}

// TransactGetItems retrieves items transactionally.
func (s *TableStore) TransactGetItems(items []transactGetItem) ([]transactGetResponse, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	responses := make([]transactGetResponse, len(items))
	for i, txItem := range items {
		if txItem.Get == nil {
			continue
		}
		g := txItem.Get
		table, awsErr := s.getTable(g.TableName)
		if awsErr != nil {
			return nil, awsErr
		}
		for _, item := range table.Items {
			if table.keyMatchesItem(g.Key, item) {
				result := copyItem(item)
				if g.ProjectionExpression != "" {
					result = applyProjection(result, g.ProjectionExpression, g.ExpressionAttributeNames)
				}
				responses[i] = transactGetResponse{Item: result}
				break
			}
		}
	}
	return responses, nil
}

func formatReasons(reasons []cancellationReason) string {
	result := ""
	for i, r := range reasons {
		if i > 0 {
			result += ", "
		}
		result += r.Code
	}
	return result
}

// copyItem returns a shallow copy of an item.
func copyItem(item Item) Item {
	if item == nil {
		return nil
	}
	result := make(Item, len(item))
	for k, v := range item {
		result[k] = v
	}
	return result
}
