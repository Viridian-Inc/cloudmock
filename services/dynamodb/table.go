package dynamodb

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// AttributeValue is the DynamoDB typed value format.
// We store it as map[string]any matching the JSON wire format.
// e.g., {"S": "hello"}, {"N": "42"}, {"BOOL": true}
type AttributeValue = map[string]any

// Item is a DynamoDB item: a map of attribute names to typed values.
type Item = map[string]AttributeValue

// KeySchemaElement describes a single element of the table's key schema.
type KeySchemaElement struct {
	AttributeName string `json:"AttributeName"`
	KeyType       string `json:"KeyType"` // HASH or RANGE
}

// AttributeDefinition describes the type of a key attribute.
type AttributeDefinition struct {
	AttributeName string `json:"AttributeName"`
	AttributeType string `json:"AttributeType"` // S, N, or B
}

// ProvisionedThroughput holds the read/write capacity for a table.
type ProvisionedThroughput struct {
	ReadCapacityUnits  int64 `json:"ReadCapacityUnits"`
	WriteCapacityUnits int64 `json:"WriteCapacityUnits"`
}

// GSI represents a Global Secondary Index definition.
type GSI struct {
	IndexName             string                 `json:"IndexName"`
	KeySchema             []KeySchemaElement     `json:"KeySchema"`
	Projection            map[string]any         `json:"Projection"`
	ProvisionedThroughput *ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
}

// LSI represents a Local Secondary Index definition.
type LSI struct {
	IndexName  string             `json:"IndexName"`
	KeySchema  []KeySchemaElement `json:"KeySchema"`
	Projection map[string]any     `json:"Projection"`
}

// Table is the in-memory representation of a DynamoDB table.
type Table struct {
	mu                    sync.RWMutex // per-table lock for item operations
	Name                  string
	KeySchema             []KeySchemaElement
	AttributeDefinitions  []AttributeDefinition
	Status                string  // ACTIVE, CREATING, DELETING
	CreationDateTime      float64 // Unix timestamp
	BillingMode           string
	ProvisionedThroughput *ProvisionedThroughput
	GSIs                  []GSI
	LSIs                  []LSI
	Stream                *Stream           // nil if streams not enabled
	TTL                   *TTLSpecification // nil if TTL not configured

	// ResourcePolicy holds the table's resource-based policy JSON (set via PutResourcePolicy).
	ResourcePolicy           string
	ResourcePolicyRevisionID string

	// Tags holds the table's resource tags (key/value pairs).
	Tags map[string]string

	// Cached key names (set once at table creation, read-only thereafter).
	cachedHashKey  string
	cachedRangeKey string

	// --- New partition-based storage ---
	partitions map[string]*Partition  // pkValue → partition
	gsiStores  map[string]*IndexStore // indexName → index store
	lsiStores  map[string]*IndexStore // indexName → index store
	count      atomic.Int64

	// --- Backward-compat shims for store.go / handlers.go (Task 4 removes these) ---
	Items    []Item            // Deprecated: kept for store.go compatibility
	ItemCount int64            // Deprecated: kept for handlers.go compatibility
	GSIItems map[string][]Item // Deprecated: kept for handlers.go / store.go compatibility
	LSIItems map[string][]Item // Deprecated: kept for handlers.go / store.go compatibility
}

// initPartitions initializes the partition-based storage for a table.
// Must be called after KeySchema, GSIs, and LSIs are set.
func (t *Table) initPartitions() {
	// Cache key names so hashKeyName()/rangeKeyName() are O(1).
	for _, ks := range t.KeySchema {
		if ks.KeyType == "HASH" {
			t.cachedHashKey = ks.AttributeName
		} else if ks.KeyType == "RANGE" {
			t.cachedRangeKey = ks.AttributeName
		}
	}

	t.partitions = make(map[string]*Partition)
	t.gsiStores = make(map[string]*IndexStore)
	t.lsiStores = make(map[string]*IndexStore)

	for _, gsi := range t.GSIs {
		sk := gsiRangeKeyName(gsi.KeySchema)
		t.gsiStores[gsi.IndexName] = newIndexStore(sk)
	}
	for _, lsi := range t.LSIs {
		sk := gsiRangeKeyName(lsi.KeySchema)
		t.lsiStores[lsi.IndexName] = newIndexStore(sk)
	}
}

// hashKeyName returns the attribute name of the HASH key.
func (t *Table) hashKeyName() string {
	return t.cachedHashKey
}

// rangeKeyName returns the attribute name of the RANGE key, or "" if none.
func (t *Table) rangeKeyName() string {
	return t.cachedRangeKey
}

// keyMatchesItem returns true if the given key map matches the item's key attributes.
func (t *Table) keyMatchesItem(key Item, item Item) bool {
	hk := t.hashKeyName()
	if !avEqual(key[hk], item[hk]) {
		return false
	}
	rk := t.rangeKeyName()
	if rk != "" {
		if !avEqual(key[rk], item[rk]) {
			return false
		}
	}
	return true
}

// gsiHashKeyName returns the HASH key name for the given GSI.
func gsiHashKeyName(ks []KeySchemaElement) string {
	for _, k := range ks {
		if k.KeyType == "HASH" {
			return k.AttributeName
		}
	}
	return ""
}

// gsiRangeKeyName returns the RANGE key name for the given GSI, or "".
func gsiRangeKeyName(ks []KeySchemaElement) string {
	for _, k := range ks {
		if k.KeyType == "RANGE" {
			return k.AttributeName
		}
	}
	return ""
}

// indexItem adds an item to all applicable GSI and LSI indexes.
func (t *Table) indexItem(item Item) {
	for _, gsi := range t.GSIs {
		hk := gsiHashKeyName(gsi.KeySchema)
		if hk == "" {
			continue
		}
		if _, ok := item[hk]; !ok {
			continue // item doesn't have the GSI's hash key
		}
		rk := gsiRangeKeyName(gsi.KeySchema)
		if rk != "" {
			if _, ok := item[rk]; !ok {
				continue // item doesn't have the GSI's range key
			}
		}
		// Remove any existing item with same GSI key, then add.
		if store, ok := t.gsiStores[gsi.IndexName]; ok {
			store.remove(item, hk)
			store.put(item, hk)
		}
		// Also maintain legacy GSIItems for backward compat.
		t.removeFromIndex(t.GSIItems, gsi.IndexName, gsi.KeySchema, item)
		t.GSIItems[gsi.IndexName] = append(t.GSIItems[gsi.IndexName], item)
	}
	for _, lsi := range t.LSIs {
		hk := gsiHashKeyName(lsi.KeySchema)
		if hk == "" {
			continue
		}
		if _, ok := item[hk]; !ok {
			continue
		}
		rk := gsiRangeKeyName(lsi.KeySchema)
		if rk != "" {
			if _, ok := item[rk]; !ok {
				continue
			}
		}
		if store, ok := t.lsiStores[lsi.IndexName]; ok {
			store.remove(item, hk)
			store.put(item, hk)
		}
		t.removeFromIndex(t.LSIItems, lsi.IndexName, lsi.KeySchema, item)
		t.LSIItems[lsi.IndexName] = append(t.LSIItems[lsi.IndexName], item)
	}
}

// deindexItem removes an item from all GSI and LSI indexes.
func (t *Table) deindexItem(item Item) {
	for _, gsi := range t.GSIs {
		hk := gsiHashKeyName(gsi.KeySchema)
		if store, ok := t.gsiStores[gsi.IndexName]; ok {
			store.remove(item, hk)
		}
		t.removeFromIndex(t.GSIItems, gsi.IndexName, gsi.KeySchema, item)
	}
	for _, lsi := range t.LSIs {
		hk := gsiHashKeyName(lsi.KeySchema)
		if store, ok := t.lsiStores[lsi.IndexName]; ok {
			store.remove(item, hk)
		}
		t.removeFromIndex(t.LSIItems, lsi.IndexName, lsi.KeySchema, item)
	}
}

// removeFromIndex removes an item from a specific legacy index by matching key schema.
func (t *Table) removeFromIndex(indexItems map[string][]Item, indexName string, ks []KeySchemaElement, item Item) {
	items := indexItems[indexName]
	hk := gsiHashKeyName(ks)
	rk := gsiRangeKeyName(ks)
	for i, existing := range items {
		if !avEqual(item[hk], existing[hk]) {
			continue
		}
		if rk != "" && !avEqual(item[rk], existing[rk]) {
			continue
		}
		indexItems[indexName] = append(items[:i], items[i+1:]...)
		return
	}
}

// avEqual compares two AttributeValues for equality.
func avEqual(a, b AttributeValue) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for _, typ := range []string{"S", "N", "B", "BOOL", "NULL"} {
		va, oka := a[typ]
		vb, okb := b[typ]
		if oka && okb {
			return fmt.Sprint(va) == fmt.Sprint(vb)
		}
		if oka != okb {
			return false
		}
	}
	return false
}

// --- New partition-based methods ---

// getOrCreatePartition returns the partition for a given hash key value,
// creating it if it doesn't exist. Caller must hold table.mu at least as RLock
// for reading, or Lock for creating.
func (t *Table) getOrCreatePartition(pkVal string) *Partition {
	part, ok := t.partitions[pkVal]
	if !ok {
		part = newPartition(t.cachedRangeKey)
		t.partitions[pkVal] = part
	}
	return part
}

// getPartition returns the partition for a given hash key value, or nil if not found.
func (t *Table) getPartition(pkVal string) *Partition {
	return t.partitions[pkVal]
}

// putItem inserts or replaces an item. Returns the previous item (nil if new).
func (t *Table) putItem(item Item) Item {
	hk := t.hashKeyName()
	rk := t.rangeKeyName()
	pkVal := attrString(item[hk])

	part, ok := t.partitions[pkVal]
	if !ok {
		part = newPartition(rk)
		t.partitions[pkVal] = part
	}

	old, replaced := part.put(item)
	if !replaced {
		t.count.Add(1)
	}
	return old
}

// getItem retrieves an item by its key. Returns (item, true) if found.
func (t *Table) getItem(key Item) (Item, bool) {
	hk := t.hashKeyName()
	pkVal := attrString(key[hk])

	part, ok := t.partitions[pkVal]
	if !ok {
		return nil, false
	}
	return part.get(key)
}

// deleteItem removes an item by key. Returns the old item (nil if not found).
func (t *Table) deleteItem(key Item) Item {
	hk := t.hashKeyName()
	pkVal := attrString(key[hk])

	part, ok := t.partitions[pkVal]
	if !ok {
		return nil
	}

	old, deleted := part.delete(key)
	if deleted {
		t.count.Add(-1)
		if part.len() == 0 {
			delete(t.partitions, pkVal)
		}
	}
	return old
}

// queryPartition returns items from a single partition, sorted by sort key.
// If startKey is nil, returns from the beginning (or end if descending).
func (t *Table) queryPartition(pkValue string, startKey Item, ascending bool, limit int) []Item {
	part, ok := t.partitions[pkValue]
	if !ok {
		return nil
	}
	return part.scan(ascending, limit)
}

// scanAll returns all items across all partitions.
// If limit > 0, at most limit items are returned.
func (t *Table) scanAll(limit int) []Item {
	var result []Item
	for _, part := range t.partitions {
		items := part.scan(true, 0)
		result = append(result, items...)
		if limit > 0 && len(result) >= limit {
			return result[:limit]
		}
	}
	return result
}

// allItems returns all items (convenience for TTL reaper and other legacy code).
func (t *Table) allItems() []Item {
	return t.scanAll(0)
}

// itemCount returns the total number of items in the table.
func (t *Table) itemCount() int64 {
	return t.count.Load()
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
