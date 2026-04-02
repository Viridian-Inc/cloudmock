package dynamodb

import (
	"fmt"
	"sync"
	"testing"
)

func newTestStore() *TableStore {
	return NewTableStore("123456789012", "us-east-1")
}

func createTestTable(t *testing.T, store *TableStore, name string) {
	t.Helper()
	_, err := store.CreateTable(
		name,
		[]KeySchemaElement{
			{AttributeName: "pk", KeyType: "HASH"},
			{AttributeName: "sk", KeyType: "RANGE"},
		},
		[]AttributeDefinition{
			{AttributeName: "pk", AttributeType: "S"},
			{AttributeName: "sk", AttributeType: "S"},
		},
		"PAY_PER_REQUEST",
		nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("CreateTable(%s): %v", name, err)
	}
}

func makeItem(pk, sk string) Item {
	return Item{
		"pk": {"S": pk},
		"sk": {"S": sk},
	}
}

func TestStore_CreateAndDescribe(t *testing.T) {
	store := newTestStore()
	createTestTable(t, store, "Users")

	table, awsErr := store.DescribeTable("Users")
	if awsErr != nil {
		t.Fatalf("DescribeTable: %v", awsErr)
	}
	if table.Name != "Users" {
		t.Errorf("expected table name Users, got %s", table.Name)
	}
	if table.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %s", table.Status)
	}
	if len(table.KeySchema) != 2 {
		t.Errorf("expected 2 key schema elements, got %d", len(table.KeySchema))
	}

	// Describe non-existent table.
	_, awsErr = store.DescribeTable("NonExistent")
	if awsErr == nil {
		t.Fatal("expected error describing non-existent table")
	}

	// Duplicate create.
	_, awsErr = store.CreateTable("Users",
		[]KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		[]AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		"", nil, nil, nil, nil,
	)
	if awsErr == nil {
		t.Fatal("expected error creating duplicate table")
	}
}

func TestStore_PutGetDelete(t *testing.T) {
	store := newTestStore()
	createTestTable(t, store, "Items")

	item := makeItem("user1", "profile")
	item["name"] = AttributeValue{"S": "Alice"}

	// Put
	if err := store.PutItem("Items", item); err != nil {
		t.Fatalf("PutItem: %v", err)
	}

	// Get
	got, awsErr := store.GetItem("Items", makeItem("user1", "profile"), "", nil)
	if awsErr != nil {
		t.Fatalf("GetItem: %v", awsErr)
	}
	if got == nil {
		t.Fatal("expected item, got nil")
	}
	if fmt.Sprint(got["name"]["S"]) != "Alice" {
		t.Errorf("expected name Alice, got %v", got["name"])
	}

	// Get non-existent
	got, awsErr = store.GetItem("Items", makeItem("user1", "other"), "", nil)
	if awsErr != nil {
		t.Fatalf("GetItem: %v", awsErr)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent item, got %v", got)
	}

	// Overwrite
	item2 := makeItem("user1", "profile")
	item2["name"] = AttributeValue{"S": "Bob"}
	if err := store.PutItem("Items", item2); err != nil {
		t.Fatalf("PutItem overwrite: %v", err)
	}
	got, _ = store.GetItem("Items", makeItem("user1", "profile"), "", nil)
	if fmt.Sprint(got["name"]["S"]) != "Bob" {
		t.Errorf("expected name Bob after overwrite, got %v", got["name"])
	}

	// Delete
	if err := store.DeleteItem("Items", makeItem("user1", "profile")); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	got, _ = store.GetItem("Items", makeItem("user1", "profile"), "", nil)
	if got != nil {
		t.Error("expected nil after delete")
	}

	// Delete non-existent (should not error)
	if err := store.DeleteItem("Items", makeItem("user1", "profile")); err != nil {
		t.Fatalf("DeleteItem non-existent: %v", err)
	}

	// Put/Get/Delete on non-existent table
	if err := store.PutItem("NoTable", item); err == nil {
		t.Fatal("expected error on non-existent table")
	}
	if _, err := store.GetItem("NoTable", item, "", nil); err == nil {
		t.Fatal("expected error on non-existent table")
	}
	if err := store.DeleteItem("NoTable", item); err == nil {
		t.Fatal("expected error on non-existent table")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := newTestStore()
	createTestTable(t, store, "Concurrent")

	const goroutines = 1000
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			item := makeItem("pk", fmt.Sprintf("sk-%04d", n))
			item["val"] = AttributeValue{"N": fmt.Sprintf("%d", n)}
			if err := store.PutItem("Concurrent", item); err != nil {
				t.Errorf("goroutine %d PutItem: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all items were written.
	items, count, scanned, awsErr := store.Scan("Concurrent", "", "", nil, nil, 0)
	if awsErr != nil {
		t.Fatalf("Scan: %v", awsErr)
	}
	if count != goroutines {
		t.Errorf("expected %d items, got %d (scanned=%d)", goroutines, count, scanned)
	}
	if len(items) != goroutines {
		t.Errorf("expected %d items in result, got %d", goroutines, len(items))
	}

	// Concurrent reads.
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			key := makeItem("pk", fmt.Sprintf("sk-%04d", n))
			got, err := store.GetItem("Concurrent", key, "", nil)
			if err != nil {
				t.Errorf("goroutine %d GetItem: %v", n, err)
			}
			if got == nil {
				t.Errorf("goroutine %d: expected item, got nil", n)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent deletes.
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			key := makeItem("pk", fmt.Sprintf("sk-%04d", n))
			if err := store.DeleteItem("Concurrent", key); err != nil {
				t.Errorf("goroutine %d DeleteItem: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	items, count, _, _ = store.Scan("Concurrent", "", "", nil, nil, 0)
	if count != 0 {
		t.Errorf("expected 0 items after delete, got %d", count)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items in result after delete, got %d", len(items))
	}
}

func BenchmarkStore_GetItem_Concurrent(b *testing.B) {
	store := newTestStore()
	_, _ = store.CreateTable(
		"Bench",
		[]KeySchemaElement{
			{AttributeName: "pk", KeyType: "HASH"},
			{AttributeName: "sk", KeyType: "RANGE"},
		},
		[]AttributeDefinition{
			{AttributeName: "pk", AttributeType: "S"},
			{AttributeName: "sk", AttributeType: "S"},
		},
		"PAY_PER_REQUEST",
		nil, nil, nil, nil,
	)

	// Pre-populate 100K items.
	const numItems = 100_000
	for i := 0; i < numItems; i++ {
		item := makeItem(fmt.Sprintf("pk-%05d", i/100), fmt.Sprintf("sk-%05d", i%100))
		item["data"] = AttributeValue{"S": fmt.Sprintf("value-%d", i)}
		store.PutItem("Bench", item)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			idx := i % numItems
			key := makeItem(fmt.Sprintf("pk-%05d", idx/100), fmt.Sprintf("sk-%05d", idx%100))
			store.GetItem("Bench", key, "", nil)
			i++
		}
	})
}
