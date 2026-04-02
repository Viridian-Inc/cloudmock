package dynamodb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestTable creates a table with the given key schema for testing.
func newTestTable(hashKey string, rangeKey string) *Table {
	ks := []KeySchemaElement{{AttributeName: hashKey, KeyType: "HASH"}}
	if rangeKey != "" {
		ks = append(ks, KeySchemaElement{AttributeName: rangeKey, KeyType: "RANGE"})
	}
	t := &Table{
		Name:      "test-table",
		KeySchema: ks,
		Status:    "ACTIVE",
	}
	t.initPartitions()
	return t
}

func TestTable_PutAndGet_HashOnly(t *testing.T) {
	tbl := newTestTable("pk", "")

	item := Item{
		"pk":   AttributeValue{"S": "user1"},
		"name": AttributeValue{"S": "Alice"},
	}
	old := tbl.putItem(item)
	assert.Nil(t, old, "first put should not return old item")

	got, found := tbl.getItem(Item{"pk": AttributeValue{"S": "user1"}})
	require.True(t, found)
	assert.Equal(t, "Alice", got["name"]["S"])
}

func TestTable_PutAndGet_Composite(t *testing.T) {
	tbl := newTestTable("pk", "sk")

	item := Item{
		"pk":   AttributeValue{"S": "user1"},
		"sk":   AttributeValue{"S": "profile"},
		"name": AttributeValue{"S": "Alice"},
	}
	old := tbl.putItem(item)
	assert.Nil(t, old)

	got, found := tbl.getItem(Item{
		"pk": AttributeValue{"S": "user1"},
		"sk": AttributeValue{"S": "profile"},
	})
	require.True(t, found)
	assert.Equal(t, "Alice", got["name"]["S"])

	// Different sort key should not be found.
	_, found = tbl.getItem(Item{
		"pk": AttributeValue{"S": "user1"},
		"sk": AttributeValue{"S": "settings"},
	})
	assert.False(t, found)
}

func TestTable_PutOverwrite(t *testing.T) {
	tbl := newTestTable("pk", "sk")

	item1 := Item{
		"pk":   AttributeValue{"S": "user1"},
		"sk":   AttributeValue{"S": "profile"},
		"name": AttributeValue{"S": "Alice"},
	}
	tbl.putItem(item1)

	item2 := Item{
		"pk":   AttributeValue{"S": "user1"},
		"sk":   AttributeValue{"S": "profile"},
		"name": AttributeValue{"S": "Bob"},
	}
	old := tbl.putItem(item2)
	require.NotNil(t, old, "overwrite should return old item")
	assert.Equal(t, "Alice", old["name"]["S"])

	assert.Equal(t, int64(1), tbl.itemCount(), "count should still be 1 after overwrite")

	got, found := tbl.getItem(Item{
		"pk": AttributeValue{"S": "user1"},
		"sk": AttributeValue{"S": "profile"},
	})
	require.True(t, found)
	assert.Equal(t, "Bob", got["name"]["S"])
}

func TestTable_Delete(t *testing.T) {
	tbl := newTestTable("pk", "sk")

	item := Item{
		"pk":   AttributeValue{"S": "user1"},
		"sk":   AttributeValue{"S": "profile"},
		"name": AttributeValue{"S": "Alice"},
	}
	tbl.putItem(item)
	assert.Equal(t, int64(1), tbl.itemCount())

	old := tbl.deleteItem(Item{
		"pk": AttributeValue{"S": "user1"},
		"sk": AttributeValue{"S": "profile"},
	})
	require.NotNil(t, old)
	assert.Equal(t, "Alice", old["name"]["S"])
	assert.Equal(t, int64(0), tbl.itemCount())

	_, found := tbl.getItem(Item{
		"pk": AttributeValue{"S": "user1"},
		"sk": AttributeValue{"S": "profile"},
	})
	assert.False(t, found)

	// Delete non-existent item returns nil.
	old = tbl.deleteItem(Item{
		"pk": AttributeValue{"S": "nonexistent"},
		"sk": AttributeValue{"S": "nope"},
	})
	assert.Nil(t, old)
}

func TestTable_QueryPartition(t *testing.T) {
	tbl := newTestTable("pk", "sk")

	// Insert 100 items in same partition with different sort keys.
	for i := 0; i < 100; i++ {
		tbl.putItem(Item{
			"pk":  AttributeValue{"S": "partition1"},
			"sk":  AttributeValue{"S": fmt.Sprintf("item-%03d", i)},
			"val": AttributeValue{"N": fmt.Sprintf("%d", i)},
		})
	}

	// Query ascending, no limit.
	results := tbl.queryPartition("S:partition1", nil, true, 0)
	require.Len(t, results, 100)

	// Verify ascending order.
	for i := 0; i < 99; i++ {
		skA := fmt.Sprint(results[i]["sk"]["S"])
		skB := fmt.Sprint(results[i+1]["sk"]["S"])
		assert.True(t, skA < skB, "expected %s < %s", skA, skB)
	}
}

func TestTable_QueryPartition_Descending(t *testing.T) {
	tbl := newTestTable("pk", "sk")

	for i := 0; i < 50; i++ {
		tbl.putItem(Item{
			"pk": AttributeValue{"S": "p1"},
			"sk": AttributeValue{"S": fmt.Sprintf("sk-%03d", i)},
		})
	}

	// Query descending with limit 10.
	results := tbl.queryPartition("S:p1", nil, false, 10)
	require.Len(t, results, 10)

	// Verify descending order.
	for i := 0; i < 9; i++ {
		skA := fmt.Sprint(results[i]["sk"]["S"])
		skB := fmt.Sprint(results[i+1]["sk"]["S"])
		assert.True(t, skA > skB, "expected %s > %s", skA, skB)
	}
}

func TestTable_ItemCount(t *testing.T) {
	tbl := newTestTable("pk", "sk")

	for i := 0; i < 1000; i++ {
		tbl.putItem(Item{
			"pk": AttributeValue{"S": fmt.Sprintf("pk-%d", i%10)},
			"sk": AttributeValue{"S": fmt.Sprintf("sk-%03d", i)},
		})
	}
	assert.Equal(t, int64(1000), tbl.itemCount())
}

func BenchmarkTable_GetItem_1M(b *testing.B) {
	tbl := newTestTable("pk", "sk")

	// Insert 1M items across 1000 partitions.
	for i := 0; i < 1_000_000; i++ {
		tbl.putItem(Item{
			"pk": AttributeValue{"S": fmt.Sprintf("pk-%04d", i%1000)},
			"sk": AttributeValue{"S": fmt.Sprintf("sk-%06d", i)},
		})
	}

	key := Item{
		"pk": AttributeValue{"S": "pk-0500"},
		"sk": AttributeValue{"S": "sk-500500"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.getItem(key)
	}
}

func BenchmarkTable_PutItem(b *testing.B) {
	tbl := newTestTable("pk", "sk")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.putItem(Item{
			"pk": AttributeValue{"S": fmt.Sprintf("pk-%04d", i%1000)},
			"sk": AttributeValue{"S": fmt.Sprintf("sk-%06d", i)},
		})
	}
}

func BenchmarkTable_QueryPartition_100(b *testing.B) {
	tbl := newTestTable("pk", "sk")

	// Insert 10K items across 100 partitions (100 items per partition).
	for i := 0; i < 10_000; i++ {
		tbl.putItem(Item{
			"pk": AttributeValue{"S": fmt.Sprintf("pk-%02d", i%100)},
			"sk": AttributeValue{"S": fmt.Sprintf("sk-%04d", i)},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.queryPartition("S:pk-50", nil, true, 100)
	}
}
