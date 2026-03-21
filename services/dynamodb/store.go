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
func (s *TableStore) CreateTable(name string, keySchema []KeySchemaElement, attrDefs []AttributeDefinition, billingMode string, pt *ProvisionedThroughput) (*Table, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tables[name]; ok {
		return nil, service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Table already exists: %s", name), http.StatusBadRequest)
	}

	if billingMode == "" {
		billingMode = "PROVISIONED"
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

	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	// Replace existing item with same key, or append.
	for i, existing := range table.Items {
		if table.keyMatchesItem(item, existing) {
			table.Items[i] = item
			return nil
		}
	}
	table.Items = append(table.Items, item)
	table.ItemCount = int64(len(table.Items))
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

	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return awsErr
	}

	for i, item := range table.Items {
		if table.keyMatchesItem(key, item) {
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
		table.Items[idx] = target
	} else {
		table.Items = append(table.Items, target)
		table.ItemCount = int64(len(table.Items))
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

// Query finds items matching a key condition expression, applies filter and projection.
func (s *TableStore) Query(tableName string, keyCondExpr string, filterExpr string, projExpr string, exprNames map[string]string, exprValues map[string]AttributeValue, scanForward *bool, limit int) ([]Item, int, int, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table, awsErr := s.getTable(tableName)
	if awsErr != nil {
		return nil, 0, 0, awsErr
	}

	// Find items matching key condition.
	var matched []Item
	for _, item := range table.Items {
		if evaluateCondition(keyCondExpr, item, exprNames, exprValues) {
			matched = append(matched, item)
		}
	}

	scannedCount := len(matched)

	// Sort by sort key if present.
	rk := table.rangeKeyName()
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
