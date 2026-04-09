package dynamodb

import (
	"fmt"
	"strings"
	"sync"

	gojson "github.com/goccy/go-json"
	"github.com/tidwall/btree"
)

// frozenGetItemResponse is the JSON wrapper for pre-serialized GetItem responses.
type frozenGetItemResponse struct {
	Item Item `json:"Item"`
}

// pkPool interns partition key strings to avoid repeat allocations.
// DynamoDB workloads repeatedly access the same partition keys, so caching
// the "S:value" string avoids a new allocation on every request.
// Uses sync.Map for lock-free reads on the hot path.
var pkPool sync.Map // string → string (interned "TYPE:value")

// attrString converts an AttributeValue to a comparable string representation.
// The format is "TYPE:value" to ensure different types don't collide.
// Uses string interning for the common string-key case.
func attrString(av AttributeValue) string {
	if av == nil {
		return ""
	}
	if v, ok := av["S"]; ok {
		if s, ok := v.(string); ok {
			// Hot path: check intern pool first (zero-alloc on cache hit).
			if interned, ok := pkPool.Load(s); ok {
				return interned.(string)
			}
			result := "S:" + s
			pkPool.Store(s, result)
			return result
		}
		return "S:" + fmt.Sprint(v)
	}
	if v, ok := av["N"]; ok {
		var s string
		if str, ok := v.(string); ok {
			s = str
		} else {
			s = fmt.Sprint(v)
		}
		return "N:" + padNumber(s)
	}
	if v, ok := av["B"]; ok {
		if s, ok := v.(string); ok {
			return "B:" + s
		}
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
//
// Frozen JSON cache: each item's GetItem JSON response is pre-serialized at write time
// and stored in frozenJSON. GetItemRaw returns the cached bytes with zero marshaling,
// eliminating the #1 CPU bottleneck (28% of CPU was spent in gojson.Marshal on reads).
type Partition struct {
	mu         sync.RWMutex        // per-partition lock for item operations
	single     Item                // used when there is no sort key
	singleJSON []byte              // pre-serialized JSON for single item
	tree       *btree.BTreeG[Item] // used when there is a sort key
	frozenJSON map[string][]byte   // skValue → pre-serialized GetItem JSON response
	skName     string              // sort key attribute name (empty if no sort key)
	count      int
}

// newPartition creates a Partition for the given sort key name.
// If skName is empty, the partition stores a single item (hash-only table).
func newPartition(skName string) *Partition {
	p := &Partition{skName: skName}
	if skName != "" {
		p.tree = btree.NewBTreeG(func(a, b Item) bool {
			return sortKeyValue(a, skName) < sortKeyValue(b, skName)
		})
		p.frozenJSON = make(map[string][]byte)
	}
	return p
}

// put inserts or replaces an item. Returns the old item and whether it replaced.
// Also pre-serializes the item to JSON for the frozen cache (zero-marshal reads).
func (p *Partition) put(item Item) (old Item, replaced bool) {
	// Pre-serialize for frozen cache. Errors are silently ignored (read will fall back to marshal).
	frozen, _ := gojson.Marshal(frozenGetItemResponse{Item: item})

	if p.skName == "" {
		old = p.single
		replaced = old != nil
		p.single = item
		p.singleJSON = frozen
		if !replaced {
			p.count++
		}
		return old, replaced
	}
	old, replaced = p.tree.Set(item)
	if !replaced {
		p.count++
	}
	// Cache frozen JSON keyed by sort key value.
	skVal := sortKeyValue(item, p.skName)
	p.frozenJSON[skVal] = frozen
	return old, replaced
}

// getRaw returns the pre-serialized JSON for an item, or nil if not cached.
// This is the zero-marshal fast path for GetItemRaw.
func (p *Partition) getRaw(sortKey Item) []byte {
	if p.skName == "" {
		return p.singleJSON
	}
	skVal := sortKeyValue(sortKey, p.skName)
	return p.frozenJSON[skVal]
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
		p.singleJSON = nil
		p.count--
		return old, true
	}
	old, deleted = p.tree.Delete(sortKey)
	if deleted {
		p.count--
		skVal := sortKeyValue(sortKey, p.skName)
		delete(p.frozenJSON, skVal)
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
