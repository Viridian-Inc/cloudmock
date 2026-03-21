package dynamodb

import "fmt"

// AttributeValue is the DynamoDB typed value format.
// We store it as map[string]interface{} matching the JSON wire format.
// e.g., {"S": "hello"}, {"N": "42"}, {"BOOL": true}
type AttributeValue = map[string]interface{}

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
	KeySchema             []KeySchemaElement      `json:"KeySchema"`
	Projection            map[string]interface{} `json:"Projection"`
	ProvisionedThroughput *ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
}

// LSI represents a Local Secondary Index definition.
type LSI struct {
	IndexName  string                 `json:"IndexName"`
	KeySchema  []KeySchemaElement      `json:"KeySchema"`
	Projection map[string]interface{} `json:"Projection"`
}

// Table is the in-memory representation of a DynamoDB table.
type Table struct {
	Name                  string
	KeySchema             []KeySchemaElement
	AttributeDefinitions  []AttributeDefinition
	Items                 []Item
	Status                string  // ACTIVE, CREATING, DELETING
	CreationDateTime      float64 // Unix timestamp
	ItemCount             int64
	BillingMode           string
	ProvisionedThroughput *ProvisionedThroughput
	GSIs                  []GSI
	LSIs                  []LSI
	GSIItems              map[string][]Item // indexName → items
	LSIItems              map[string][]Item // indexName → items
}

// hashKeyName returns the attribute name of the HASH key.
func (t *Table) hashKeyName() string {
	for _, ks := range t.KeySchema {
		if ks.KeyType == "HASH" {
			return ks.AttributeName
		}
	}
	return ""
}

// rangeKeyName returns the attribute name of the RANGE key, or "" if none.
func (t *Table) rangeKeyName() string {
	for _, ks := range t.KeySchema {
		if ks.KeyType == "RANGE" {
			return ks.AttributeName
		}
	}
	return ""
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
		// Remove any existing item with same GSI key
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
		t.removeFromIndex(t.LSIItems, lsi.IndexName, lsi.KeySchema, item)
		t.LSIItems[lsi.IndexName] = append(t.LSIItems[lsi.IndexName], item)
	}
}

// deindexItem removes an item from all GSI and LSI indexes.
func (t *Table) deindexItem(item Item) {
	for _, gsi := range t.GSIs {
		t.removeFromIndex(t.GSIItems, gsi.IndexName, gsi.KeySchema, item)
	}
	for _, lsi := range t.LSIs {
		t.removeFromIndex(t.LSIItems, lsi.IndexName, lsi.KeySchema, item)
	}
}

// removeFromIndex removes an item from a specific index by matching key schema.
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
