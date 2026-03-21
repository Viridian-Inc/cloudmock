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
