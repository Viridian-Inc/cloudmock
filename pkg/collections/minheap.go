package collections

import "container/heap"

type MinHeap[K any, V comparable] struct {
	items  *heapItems[K, V]
	lessFn func(a, b K) bool
}

type heapItem[K any, V comparable] struct {
	key   K
	value V
	index int
}

type heapItems[K any, V comparable] struct {
	data   []*heapItem[K, V]
	lessFn func(a, b K) bool
}

func (h *heapItems[K, V]) Len() int           { return len(h.data) }
func (h *heapItems[K, V]) Less(i, j int) bool { return h.lessFn(h.data[i].key, h.data[j].key) }
func (h *heapItems[K, V]) Swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
	h.data[i].index = i
	h.data[j].index = j
}
func (h *heapItems[K, V]) Push(x any) {
	item := x.(*heapItem[K, V])
	item.index = len(h.data)
	h.data = append(h.data, item)
}
func (h *heapItems[K, V]) Pop() any {
	old := h.data
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	h.data = old[:n-1]
	return item
}

func NewMinHeap[K any, V comparable](less func(a, b K) bool) *MinHeap[K, V] {
	hi := &heapItems[K, V]{lessFn: less}
	heap.Init(hi)
	return &MinHeap[K, V]{items: hi, lessFn: less}
}

func (h *MinHeap[K, V]) Push(key K, value V) {
	heap.Push(h.items, &heapItem[K, V]{key: key, value: value})
}

func (h *MinHeap[K, V]) Pop() (K, V, bool) {
	if h.items.Len() == 0 {
		var zk K
		var zv V
		return zk, zv, false
	}
	item := heap.Pop(h.items).(*heapItem[K, V])
	return item.key, item.value, true
}

func (h *MinHeap[K, V]) Peek() (K, V, bool) {
	if h.items.Len() == 0 {
		var zk K
		var zv V
		return zk, zv, false
	}
	item := h.items.data[0]
	return item.key, item.value, true
}

func (h *MinHeap[K, V]) Len() int { return h.items.Len() }

func (h *MinHeap[K, V]) RemoveByValue(value V) bool {
	for i, item := range h.items.data {
		if item.value == value {
			heap.Remove(h.items, i)
			return true
		}
	}
	return false
}
