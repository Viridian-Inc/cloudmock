package dynamodb

import (
	"fmt"
	"strings"

	"github.com/tidwall/btree"
)

// attrString converts an AttributeValue to a comparable string representation.
// The format is "TYPE:value" to ensure different types don't collide.
func attrString(av AttributeValue) string {
	if av == nil {
		return ""
	}
	if v, ok := av["S"]; ok {
		return "S:" + fmt.Sprint(v)
	}
	if v, ok := av["N"]; ok {
		// Pad numeric values for correct lexicographic ordering.
		s := fmt.Sprint(v)
		return "N:" + padNumber(s)
	}
	if v, ok := av["B"]; ok {
		return "B:" + fmt.Sprint(v)
	}
	return ""
}

// padNumber pads a numeric string so that lexicographic order matches numeric order.
// Handles negative numbers, decimals, and integers.
func padNumber(s string) string {
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	// Split on decimal point.
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]

	// Pad integer part to 20 digits (enough for int64 range).
	const width = 20
	if len(intPart) < width {
		intPart = strings.Repeat("0", width-len(intPart)) + intPart
	}

	result := intPart
	if len(parts) == 2 {
		result += "." + parts[1]
	}

	if negative {
		// For negative numbers, complement each digit so ordering is reversed.
		var buf strings.Builder
		buf.WriteByte('-')
		for _, c := range result {
			if c >= '0' && c <= '9' {
				buf.WriteByte(byte('9' - (c - '0')))
			} else {
				buf.WriteRune(c)
			}
		}
		return buf.String()
	}
	return result
}

// sortKeyValue extracts a comparable sort key string from an item.
func sortKeyValue(item Item, name string) string {
	if name == "" {
		return ""
	}
	av, ok := item[name]
	if !ok {
		return ""
	}
	return attrString(av)
}

// serializeKey creates a unique string key by combining partition key and optional sort key.
func serializeKey(item Item, hashKeyName, rangeKeyName string) string {
	pk := attrString(item[hashKeyName])
	if rangeKeyName == "" {
		return pk
	}
	return pk + "\x00" + attrString(item[rangeKeyName])
}

// Partition holds items sharing the same partition key value.
// For tables without a sort key, it holds at most one item.
// For tables with a sort key, items are stored in a B-tree sorted by sort key.
type Partition struct {
	single Item             // used when there is no sort key
	tree   *btree.BTreeG[Item] // used when there is a sort key
	skName string           // sort key attribute name (empty if no sort key)
	count  int
}

// newPartition creates a Partition for the given sort key name.
// If skName is empty, the partition stores a single item (hash-only table).
func newPartition(skName string) *Partition {
	p := &Partition{skName: skName}
	if skName != "" {
		p.tree = btree.NewBTreeG(func(a, b Item) bool {
			return sortKeyValue(a, skName) < sortKeyValue(b, skName)
		})
	}
	return p
}

// put inserts or replaces an item. Returns the old item and whether it replaced.
func (p *Partition) put(item Item) (old Item, replaced bool) {
	if p.skName == "" {
		// Hash-only: single item per partition.
		old = p.single
		replaced = old != nil
		p.single = item
		if !replaced {
			p.count++
		}
		return old, replaced
	}
	old, replaced = p.tree.Set(item)
	if !replaced {
		p.count++
	}
	return old, replaced
}

// get retrieves an item by sort key value. For hash-only tables, sortKey is ignored.
func (p *Partition) get(sortKey Item) (Item, bool) {
	if p.skName == "" {
		if p.single == nil {
			return nil, false
		}
		return p.single, true
	}
	return p.tree.Get(sortKey)
}

// delete removes an item by sort key. Returns the old item and whether it was found.
func (p *Partition) delete(sortKey Item) (old Item, deleted bool) {
	if p.skName == "" {
		old = p.single
		if old == nil {
			return nil, false
		}
		p.single = nil
		p.count--
		return old, true
	}
	old, deleted = p.tree.Delete(sortKey)
	if deleted {
		p.count--
	}
	return old, deleted
}

// scan returns items from this partition, optionally in ascending or descending order.
// If limit <= 0, all items are returned.
func (p *Partition) scan(ascending bool, limit int) []Item {
	if p.skName == "" {
		if p.single != nil {
			return []Item{p.single}
		}
		return nil
	}

	n := p.tree.Len()
	if limit > 0 && limit < n {
		n = limit
	}
	result := make([]Item, 0, n)

	iter := func(item Item) bool {
		result = append(result, item)
		return limit <= 0 || len(result) < limit
	}

	if ascending {
		p.tree.Scan(iter)
	} else {
		p.tree.Reverse(iter)
	}
	return result
}

// len returns the number of items in this partition.
func (p *Partition) len() int {
	return p.count
}

// IndexStore mirrors the main table's partition structure for a secondary index.
type IndexStore struct {
	partitions map[string]*Partition
	skName     string // sort key name for this index
}

// newIndexStore creates an IndexStore for the given sort key name.
func newIndexStore(skName string) *IndexStore {
	return &IndexStore{
		partitions: make(map[string]*Partition),
		skName:     skName,
	}
}

// put adds an item to the index store.
func (idx *IndexStore) put(item Item, hkName string) {
	pkVal := attrString(item[hkName])
	part, ok := idx.partitions[pkVal]
	if !ok {
		part = newPartition(idx.skName)
		idx.partitions[pkVal] = part
	}
	part.put(item)
}

// remove removes an item from the index store.
func (idx *IndexStore) remove(item Item, hkName string) {
	pkVal := attrString(item[hkName])
	part, ok := idx.partitions[pkVal]
	if !ok {
		return
	}
	part.delete(item)
	if part.len() == 0 {
		delete(idx.partitions, pkVal)
	}
}

// allItems returns all items across all partitions in the index store.
func (idx *IndexStore) allItems() []Item {
	var result []Item
	for _, part := range idx.partitions {
		result = append(result, part.scan(true, 0)...)
	}
	return result
}

// itemCount returns the total number of items in the index store.
func (idx *IndexStore) itemCount() int {
	total := 0
	for _, part := range idx.partitions {
		total += part.len()
	}
	return total
}
