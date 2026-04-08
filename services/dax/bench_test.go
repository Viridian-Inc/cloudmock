package dax

import (
	"testing"
)

func BenchmarkCache_GetItem_Miss(b *testing.B) {
	c := NewCache(10000, 300000, 300000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetItem("table", "pk", "sk")
	}
}

func BenchmarkCache_GetItem_Hit(b *testing.B) {
	c := NewCache(10000, 300000, 300000)
	c.SetItem("table", "pk", "sk", map[string]any{"name": "Alice"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetItem("table", "pk", "sk")
	}
}

func BenchmarkCache_SetItem(b *testing.B) {
	c := NewCache(100000, 300000, 300000)
	val := map[string]any{"name": "Alice", "age": 30}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.SetItem("table", "pk", "sk", val)
	}
}

func BenchmarkCache_SetItem_WithEviction(b *testing.B) {
	c := NewCache(1000, 300000, 300000)
	val := map[string]any{"name": "Alice"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.SetItem("table", string(rune(i%2000)), "", val)
	}
}

func BenchmarkCache_GetQuery_Hit(b *testing.B) {
	c := NewCache(10000, 300000, 300000)
	results := []any{map[string]any{"pk": "1"}, map[string]any{"pk": "2"}}
	c.SetQuery("table|idx|pk=1", results)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetQuery("table|idx|pk=1")
	}
}

func BenchmarkCache_InvalidateItem(b *testing.B) {
	c := NewCache(10000, 300000, 300000)
	for i := 0; i < 1000; i++ {
		c.SetItem("table", string(rune(i)), "", "val")
	}
	c.SetQuery("table|q1", "result")
	c.SetQuery("table|q2", "result")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.InvalidateItem("table", "pk", "sk")
	}
}

func BenchmarkCache_InvalidateTable(b *testing.B) {
	c := NewCache(10000, 300000, 300000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for j := 0; j < 100; j++ {
			c.SetItem("table", string(rune(j)), "", "val")
		}
		c.SetQuery("table|q1", "result")
		b.StartTimer()
		c.InvalidateTable("table")
	}
}

func BenchmarkCache_Parallel_GetItem(b *testing.B) {
	c := NewCache(10000, 300000, 300000)
	c.SetItem("table", "pk", "sk", map[string]any{"name": "Alice"})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.GetItem("table", "pk", "sk")
		}
	})
}

func BenchmarkCache_Parallel_Mixed(b *testing.B) {
	c := NewCache(10000, 300000, 300000)
	c.SetItem("table", "pk", "sk", map[string]any{"name": "Alice"})
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				c.SetItem("table", "pk", "sk", map[string]any{"name": "Bob"})
			} else {
				c.GetItem("table", "pk", "sk")
			}
			i++
		}
	})
}
