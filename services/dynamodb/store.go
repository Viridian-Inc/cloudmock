package dynamodb

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	gojson "github.com/goccy/go-json"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// getItemResponseRaw is used by GetItemRaw to marshal while holding the lock.
type getItemResponseRaw struct {
	Item Item `json:"Item,omitempty"`
}

// TableStore manages all DynamoDB tables in memory.
type TableStore struct {
	mu        sync.Mutex // only for operations needing global serialization (e.g. UpdateTable)
	tables    sync.Map   // map[string]*Table — lock-free for hot-path lookups
	accountID string
	region    string
}

// NewTableStore creates an empty TableStore.
func NewTableStore(accountID, region string) *TableStore {
	return &TableStore{
		accountID: accountID,
		region:    region,
	}
}

func (s *TableStore) tableARN(name string) string {
	return fmt.Sprintf("arn:aws:dynamodb:%s:%s:table/%s", s.region, s.accountID, name)
}

// CreateTable creates a new table. Returns ResourceInUseException if it already exists.
func (s *TableStore) CreateTable(name string, keySchema []KeySchemaElement, attrDefs []AttributeDefinition, billingMode string, pt *ProvisionedThroughput, gsis []GSI, lsis []LSI, streamSpec *StreamSpecification) (*Table, *service.AWSError) {
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
		Status:                "ACTIVE",
		CreationDateTime:      float64(time.Now().Unix()),
		BillingMode:           billingMode,
		ProvisionedThroughput: pt,
		GSIs:                  gsis,
		LSIs:                  lsis,
		GSIItems:              gsiItems,
		LSIItems:              lsiItems,
	}

	table.initPartitions()

	if streamSpec != nil && streamSpec.StreamEnabled {
		tableARN := s.tableARN(name)
		table.Stream = newStream(tableARN, name, streamSpec.StreamViewType)
	}

	if _, loaded := s.tables.LoadOrStore(name, table); loaded {
		return nil, service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Table already exists: %s", name), http.StatusBadRequest)
	}
	return table, nil
}

// DeleteTable removes a table. Returns ResourceNotFoundException if not found.
func (s *TableStore) DeleteTable(name string) (*Table, *service.AWSError) {
	v, loaded := s.tables.LoadAndDelete(name)
	if !loaded {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Requested resource not found: Table: %s not found", name), http.StatusBadRequest)
	}
	table := v.(*Table)
	table.Status = "DELETING"
	return table, nil
}

// DescribeTable returns table metadata. Returns ResourceNotFoundException if not found.
func (s *TableStore) DescribeTable(name string) (*Table, *service.AWSError) {
	v, ok := s.tables.Load(name)
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Requested resource not found: Table: %s not found", name), http.StatusBadRequest)
	}
	return v.(*Table), nil
}

// UpdateTable updates mutable table properties: BillingMode, ProvisionedThroughput,
// AttributeDefinitions (for new GSIs), and GlobalSecondaryIndexUpdates.
// All fields are optional; only non-zero values are applied.
// The table stays ACTIVE throughout (no async state transition in CloudMock).
func (s *TableStore) UpdateTable(name, billingMode string, pt *ProvisionedThroughput, attrDefs []AttributeDefinition) (*Table, *service.AWSError) {
	table, awsErr := s.acquireTable(name)
	if awsErr != nil {
		return nil, awsErr
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	if billingMode != "" {
		table.BillingMode = billingMode
	}
	if pt != nil {
		table.ProvisionedThroughput = pt
	}
	if len(attrDefs) > 0 {
		// Merge new attribute definitions (don't duplicate existing ones).
		existing := make(map[string]bool)
		for _, a := range table.AttributeDefinitions {
			existing[a.AttributeName] = true
		}
		for _, a := range attrDefs {
			if !existing[a.AttributeName] {
				table.AttributeDefinitions = append(table.AttributeDefinitions, a)
			}
		}
	}

	return table, nil
}

// PutResourcePolicy sets a resource-based policy on a table.
func (s *TableStore) PutResourcePolicy(resourceARN, policy string) (*Table, *service.AWSError) {
	// Extract table name from ARN (arn:aws:dynamodb:region:account:table/name)
	tableName := resourceARN
	if idx := lastIndexByte(resourceARN, '/'); idx >= 0 {
		tableName = resourceARN[idx+1:]
	}

	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	table.ResourcePolicy = policy
	table.ResourcePolicyRevisionID = fmt.Sprintf("rev-%d", time.Now().UnixNano())
	return table, nil
}

// GetResourcePolicy returns the resource policy for a table.
func (s *TableStore) GetResourcePolicy(resourceARN string) (*Table, *service.AWSError) {
	tableName := resourceARN
	if idx := lastIndexByte(resourceARN, '/'); idx >= 0 {
		tableName = resourceARN[idx+1:]
	}

	return s.acquireTable(tableName)
}

// DeleteResourcePolicy removes the resource policy from a table.
func (s *TableStore) DeleteResourcePolicy(resourceARN string) *service.AWSError {
	tableName := resourceARN
	if idx := lastIndexByte(resourceARN, '/'); idx >= 0 {
		tableName = resourceARN[idx+1:]
	}

	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	table.ResourcePolicy = ""
	table.ResourcePolicyRevisionID = ""
	return nil
}

func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// ListTables returns the names of all tables.
func (s *TableStore) ListTables() []string {
	var names []string
	s.tables.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})
	sort.Strings(names)
	return names
}

// acquireTable looks up a table from the sync.Map and returns it.
// The caller is responsible for acquiring the table-level lock.
func (s *TableStore) acquireTable(name string) (*Table, *service.AWSError) {
	v, ok := s.tables.Load(name)
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Requested resource not found: Table: %s not found", name), http.StatusBadRequest)
	}
	return v.(*Table), nil
}

// PutItem adds or replaces an item in the specified table.
func (s *TableStore) PutItem(tableName string, item Item, condExpr ...string) *service.AWSError {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	hk := table.hashKeyName()
	pkVal := attrString(item[hk])

	// Lock only the target partition, not the whole table.
	table.mu.Lock()
	part := table.getOrCreatePartition(pkVal)
	table.mu.Unlock()

	part.mu.Lock()
	defer part.mu.Unlock()

	// Evaluate condition expression if provided.
	if len(condExpr) > 0 && condExpr[0] != "" {
		key := make(Item)
		key[hk] = item[hk]
		if table.rangeKeyName() != "" {
			key[table.rangeKeyName()] = item[table.rangeKeyName()]
		}
		existing, _ := part.get(key)
		if !evaluateCondition(condExpr[0], existing, nil, nil) {
			return service.NewAWSError("ConditionalCheckFailedException",
				"The conditional request failed.", 400)
		}
	}

	old, replaced := part.put(item)
	if !replaced {
		table.count.Add(1)
	}

	// Index operations need table-level lock since they touch GSI/LSI stores.
	table.mu.Lock()
	if old != nil {
		table.deindexItem(old)
	}
	table.indexItem(item)
	table.ItemCount = table.itemCount()
	table.mu.Unlock()

	if table.Stream != nil {
		if old != nil {
			table.Stream.appendRecord("MODIFY", copyItem(old), copyItem(item))
		} else {
			table.Stream.appendRecord("INSERT", nil, copyItem(item))
		}
	}
	return nil
}

// GetItem retrieves an item by key from the specified table.
func (s *TableStore) GetItem(tableName string, key Item, projExpr string, exprNames map[string]string) (Item, *service.AWSError) {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	hk := table.hashKeyName()
	pkVal := attrString(key[hk])

	table.mu.RLock()
	part := table.getPartition(pkVal)
	table.mu.RUnlock()

	if part == nil {
		return nil, nil
	}

	part.mu.RLock()
	item, ok := part.get(key)
	if !ok {
		part.mu.RUnlock()
		return nil, nil
	}
	result := copyItem(item)
	part.mu.RUnlock()

	if projExpr != "" {
		result = applyProjection(result, projExpr, exprNames)
	}
	return result, nil
}

// GetItemRaw retrieves an item and marshals it to JSON while holding the
// partition lock, eliminating the need for copyItem. Returns nil for not found.
func (s *TableStore) GetItemRaw(tableName string, key Item) ([]byte, *service.AWSError) {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	hk := table.hashKeyName()
	pkVal := attrString(key[hk])

	table.mu.RLock()
	part := table.getPartition(pkVal)
	table.mu.RUnlock()

	if part == nil {
		return nil, nil
	}

	part.mu.RLock()
	item, ok := part.get(key)
	if !ok {
		part.mu.RUnlock()
		return nil, nil
	}
	// Marshal while holding the lock — avoids copyItem allocation.
	raw, _ := gojson.Marshal(getItemResponseRaw{Item: item})
	part.mu.RUnlock()

	return raw, nil
}

// DeleteItem removes an item by key from the specified table.
func (s *TableStore) DeleteItem(tableName string, key Item, condExpr ...string) *service.AWSError {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	hk := table.hashKeyName()
	pkVal := attrString(key[hk])

	table.mu.RLock()
	part := table.getPartition(pkVal)
	table.mu.RUnlock()

	if part == nil {
		// Partition doesn't exist, nothing to delete.
		if len(condExpr) > 0 && condExpr[0] != "" {
			if !evaluateCondition(condExpr[0], nil, nil, nil) {
				return service.NewAWSError("ConditionalCheckFailedException",
					"The conditional request failed.", 400)
			}
		}
		return nil
	}

	part.mu.Lock()

	// Evaluate condition expression if provided.
	if len(condExpr) > 0 && condExpr[0] != "" {
		existing, _ := part.get(key)
		if !evaluateCondition(condExpr[0], existing, nil, nil) {
			part.mu.Unlock()
			return service.NewAWSError("ConditionalCheckFailedException",
				"The conditional request failed.", 400)
		}
	}

	old, deleted := part.delete(key)
	part.mu.Unlock()

	if deleted {
		table.count.Add(-1)
		table.mu.Lock()
		if part.len() == 0 {
			delete(table.partitions, pkVal)
		}
		table.deindexItem(old)
		table.ItemCount = table.itemCount()
		table.mu.Unlock()

		if table.Stream != nil {
			table.Stream.appendRecord("REMOVE", copyItem(old), nil)
		}
	}
	return nil
}

// deleteFromTable removes an item by key from the given table. Caller must hold table.mu write lock.
func (s *TableStore) deleteFromTable(table *Table, key Item) *service.AWSError {
	old := table.deleteItem(key)
	if old != nil {
		table.deindexItem(old)
		table.ItemCount = table.itemCount() // sync compat field
		if table.Stream != nil {
			table.Stream.appendRecord("REMOVE", copyItem(old), nil)
		}
	}
	return nil
}

// UpdateItem updates an item using an UpdateExpression. Creates the item if it doesn't exist.
func (s *TableStore) UpdateItem(tableName string, key Item, updateExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, returnValues string) (Item, *service.AWSError) {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	return s.updateInTable(table, key, updateExpr, exprNames, exprValues, returnValues)
}

// updateInTable updates an item in the given table. Caller must hold table.mu write lock.
func (s *TableStore) updateInTable(table *Table, key Item, updateExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, returnValues string) (Item, *service.AWSError) {
	existing, found := table.getItem(key)

	var target Item
	if found {
		target = copyItem(existing)
	} else {
		target = copyItem(key)
	}

	var oldImage Item
	if found {
		oldImage = copyItem(existing)
	}

	target = parseUpdateExpression(target, updateExpr, exprNames, exprValues)

	// Remove old index entries if item existed.
	if found {
		table.deindexItem(existing)
	}

	// Insert/replace using partition store.
	table.putItem(target)
	table.indexItem(target)
	table.ItemCount = table.itemCount() // sync compat field

	if table.Stream != nil {
		if found {
			table.Stream.appendRecord("MODIFY", oldImage, copyItem(target))
		} else {
			table.Stream.appendRecord("INSERT", nil, copyItem(target))
		}
	}

	switch returnValues {
	case "ALL_NEW":
		return copyItem(target), nil
	case "NONE", "":
		return nil, nil
	default:
		return nil, nil
	}
}

// putInTable puts an item in the given table. Caller must hold table.mu write lock.
func (s *TableStore) putInTable(table *Table, item Item) {
	old := table.putItem(item)
	if old != nil {
		table.deindexItem(old)
	}
	table.indexItem(item)
	table.ItemCount = table.itemCount() // sync compat field

	if table.Stream != nil {
		if old != nil {
			table.Stream.appendRecord("MODIFY", copyItem(old), copyItem(item))
		} else {
			table.Stream.appendRecord("INSERT", nil, copyItem(item))
		}
	}
}

// Query finds items matching a key condition expression, applies filter and projection.
func (s *TableStore) Query(tableName string, indexName string, keyCondExpr string, filterExpr string, projExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, scanForward *bool, limit int) ([]Item, int, int, *service.AWSError) {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, 0, 0, awsErr
	}

	table.mu.RLock()
	defer table.mu.RUnlock()

	// Resolve expression attribute names upfront.
	resolvedKeyExpr := resolveNames(keyCondExpr, exprNames)

	// Determine hash key name and range key name based on index.
	var hk, rk string
	var partitions map[string]*Partition
	if indexName != "" {
		found := false
		for _, gsi := range table.GSIs {
			if gsi.IndexName == indexName {
				if store, ok := table.gsiStores[indexName]; ok {
					partitions = store.partitions
				}
				hk = gsiHashKeyName(gsi.KeySchema)
				rk = gsiRangeKeyName(gsi.KeySchema)
				found = true
				break
			}
		}
		if !found {
			for _, lsi := range table.LSIs {
				if lsi.IndexName == indexName {
					if store, ok := table.lsiStores[indexName]; ok {
						partitions = store.partitions
					}
					hk = gsiHashKeyName(lsi.KeySchema)
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
		partitions = table.partitions
		hk = table.hashKeyName()
		rk = table.rangeKeyName()
	}

	// Fast path: extract the partition key value from the key condition expression
	// and look up the partition directly instead of scanning all items.
	var sourceItems []Item
	pkValue := extractEqualityValue(resolvedKeyExpr, hk, exprValues)
	hasSortKeyCondition := pkValue != "" && strings.Contains(strings.ToUpper(resolvedKeyExpr), " AND ")
	if pkValue != "" && partitions != nil {
		part, ok := partitions[pkValue]
		if !ok {
			return []Item{}, 0, 0, nil
		}
		forward := true
		if scanForward != nil {
			forward = *scanForward
		}
		// If PK-only condition with no filter, pass limit directly to B-tree scan
		// to avoid copying the entire partition.
		scanLimit := 0
		if !hasSortKeyCondition && filterExpr == "" && limit > 0 {
			scanLimit = limit
		}
		sourceItems = part.scan(forward, scanLimit)
	} else {
		// Fallback: full scan (e.g. if we can't parse the partition key)
		for _, part := range partitions {
			sourceItems = append(sourceItems, part.scan(true, 0)...)
		}
	}

	// Evaluate key condition on source items.
	// If we did a direct partition lookup with only a PK equality condition (no sort key condition),
	// all items in the partition match — skip expensive per-item expression evaluation.
	var matched []Item
	if pkValue != "" && !hasSortKeyCondition {
		// Direct partition lookup, PK-only condition — all items match.
		matched = sourceItems
	} else {
		for _, item := range sourceItems {
			if evaluateCondition(keyCondExpr, item, exprNames, exprValues) {
				matched = append(matched, item)
			}
		}
	}

	scannedCount := len(matched)

	// Sort by sort key if we didn't use partition direct lookup (already sorted).
	if pkValue == "" && rk != "" {
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

	// Apply projection. Skip copyItem when no projection is needed (hot path).
	var results []Item
	if projExpr != "" {
		results = make([]Item, len(filtered))
		for i, item := range filtered {
			results[i] = applyProjection(copyItem(item), projExpr, exprNames)
		}
	} else {
		results = filtered
	}

	return results, len(results), scannedCount, nil
}

// Scan iterates all items, applies filter and projection.
func (s *TableStore) Scan(tableName string, filterExpr string, projExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, limit int) ([]Item, int, int, *service.AWSError) {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, 0, 0, awsErr
	}

	table.mu.RLock()
	defer table.mu.RUnlock()

	allItems := table.scanAll(0)
	scannedCount := len(allItems)

	var filtered []Item
	for _, item := range allItems {
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
	// Collect all table names involved, then lock them in sorted order.
	tableNameSet := make(map[string]struct{})
	for _, txItem := range items {
		if txItem.Put != nil {
			tableNameSet[txItem.Put.TableName] = struct{}{}
		}
		if txItem.Delete != nil {
			tableNameSet[txItem.Delete.TableName] = struct{}{}
		}
		if txItem.Update != nil {
			tableNameSet[txItem.Update.TableName] = struct{}{}
		}
		if txItem.ConditionCheck != nil {
			tableNameSet[txItem.ConditionCheck.TableName] = struct{}{}
		}
	}

	sortedNames := make([]string, 0, len(tableNameSet))
	for name := range tableNameSet {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Resolve all tables from sync.Map.
	tables := make(map[string]*Table, len(sortedNames))
	for _, name := range sortedNames {
		table, awsErr := s.acquireTable(name)
		if awsErr != nil {
			return awsErr
		}
		tables[name] = table
	}

	// Acquire write locks on all involved tables in sorted order (deadlock prevention).
	for _, name := range sortedNames {
		tables[name].mu.Lock()
	}
	defer func() {
		for i := len(sortedNames) - 1; i >= 0; i-- {
			tables[sortedNames[i]].mu.Unlock()
		}
	}()

	// Phase 1: Validate all conditions.
	reasons := make([]cancellationReason, len(items))
	anyFailed := false

	for i, txItem := range items {
		reasons[i] = cancellationReason{Code: "None"}

		if txItem.ConditionCheck != nil {
			cc := txItem.ConditionCheck
			table := tables[cc.TableName]
			found, ok := table.getItem(cc.Key)
			if !ok || !evaluateCondition(cc.ConditionExpression, found, cc.ExpressionAttributeNames, cc.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
				anyFailed = true
			}
		}

		if txItem.Put != nil && txItem.Put.ConditionExpression != "" {
			p := txItem.Put
			table := tables[p.TableName]
			found, ok := table.getItem(p.Item)
			if !ok {
				found = make(Item)
			}
			if !evaluateCondition(p.ConditionExpression, found, p.ExpressionAttributeNames, p.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
				anyFailed = true
			}
		}

		if txItem.Delete != nil && txItem.Delete.ConditionExpression != "" {
			d := txItem.Delete
			table := tables[d.TableName]
			found, ok := table.getItem(d.Key)
			if !ok {
				found = make(Item)
			}
			if !evaluateCondition(d.ConditionExpression, found, d.ExpressionAttributeNames, d.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
				anyFailed = true
			}
		}

		if txItem.Update != nil && txItem.Update.ConditionExpression != "" {
			u := txItem.Update
			table := tables[u.TableName]
			found, ok := table.getItem(u.Key)
			if !ok {
				found = make(Item)
			}
			if !evaluateCondition(u.ConditionExpression, found, u.ExpressionAttributeNames, u.ExpressionAttributeValues) {
				reasons[i] = cancellationReason{Code: "ConditionalCheckFailed", Message: "The conditional request failed."}
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
			table := tables[txItem.Put.TableName]
			s.putInTable(table, txItem.Put.Item)
		}
		if txItem.Delete != nil {
			table := tables[txItem.Delete.TableName]
			s.deleteFromTable(table, txItem.Delete.Key)
		}
		if txItem.Update != nil {
			u := txItem.Update
			table := tables[u.TableName]
			s.updateInTable(table, u.Key, u.UpdateExpression, u.ExpressionAttributeNames, u.ExpressionAttributeValues, "NONE")
		}
	}

	return nil
}

// TransactGetItems retrieves items transactionally.
func (s *TableStore) TransactGetItems(items []transactGetItem) ([]transactGetResponse, *service.AWSError) {
	// Collect all table names involved, lock in sorted order.
	tableNameSet := make(map[string]struct{})
	for _, txItem := range items {
		if txItem.Get != nil {
			tableNameSet[txItem.Get.TableName] = struct{}{}
		}
	}

	sortedNames := make([]string, 0, len(tableNameSet))
	for name := range tableNameSet {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Resolve all tables from sync.Map.
	tables := make(map[string]*Table, len(sortedNames))
	for _, name := range sortedNames {
		table, awsErr := s.acquireTable(name)
		if awsErr != nil {
			return nil, awsErr
		}
		tables[name] = table
	}

	// Acquire read locks on all involved tables in sorted order.
	for _, name := range sortedNames {
		tables[name].mu.RLock()
	}
	defer func() {
		for i := len(sortedNames) - 1; i >= 0; i-- {
			tables[sortedNames[i]].mu.RUnlock()
		}
	}()

	responses := make([]transactGetResponse, len(items))
	for i, txItem := range items {
		if txItem.Get == nil {
			continue
		}
		g := txItem.Get
		table := tables[g.TableName]
		item, ok := table.getItem(g.Key)
		if ok {
			result := copyItem(item)
			if g.ProjectionExpression != "" {
				result = applyProjection(result, g.ProjectionExpression, g.ExpressionAttributeNames)
			}
			responses[i] = transactGetResponse{Item: result}
		}
	}
	return responses, nil
}

// UpdateTimeToLive sets or disables TTL for a table.
func (s *TableStore) UpdateTimeToLive(tableName string, spec *TTLSpecification) *service.AWSError {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	table.mu.Lock()
	defer table.mu.Unlock()

	table.TTL = spec
	return nil
}

// DescribeTimeToLive returns the TTL configuration for a table.
func (s *TableStore) DescribeTimeToLive(tableName string) (*TTLSpecification, *service.AWSError) {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	table.mu.RLock()
	defer table.mu.RUnlock()

	return table.TTL, nil
}

// GetStream returns the stream for a table, or nil if streams are not enabled.
func (s *TableStore) GetStream(tableName string) (*Stream, *service.AWSError) {
	table, awsErr := s.acquireTable(tableName)
	if awsErr != nil {
		return nil, awsErr
	}

	table.mu.RLock()
	defer table.mu.RUnlock()

	return table.Stream, nil
}

// GetStreamByARN returns the stream matching the given ARN, or nil.
func (s *TableStore) GetStreamByARN(arn string) *Stream {
	var found *Stream
	s.tables.Range(func(key, value any) bool {
		table := value.(*Table)
		if table.Stream != nil && table.Stream.arn == arn {
			found = table.Stream
			return false
		}
		return true
	})
	return found
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
