package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinHeap_PushPop(t *testing.T) {
	h := NewMinHeap[int, int](func(a, b int) bool { return a < b })
	h.Push(3, 30)
	h.Push(1, 10)
	h.Push(2, 20)

	key, val, ok := h.Pop()
	require.True(t, ok)
	assert.Equal(t, 1, key)
	assert.Equal(t, 10, val)

	key, val, ok = h.Pop()
	require.True(t, ok)
	assert.Equal(t, 2, key)
	assert.Equal(t, 20, val)
}

func TestMinHeap_Peek(t *testing.T) {
	h := NewMinHeap[int, string](func(a, b int) bool { return a < b })
	_, _, ok := h.Peek()
	assert.False(t, ok)

	h.Push(5, "five")
	h.Push(2, "two")

	key, val, ok := h.Peek()
	require.True(t, ok)
	assert.Equal(t, 2, key)
	assert.Equal(t, "two", val)
	assert.Equal(t, 2, h.Len())
}

func TestMinHeap_Empty(t *testing.T) {
	h := NewMinHeap[int, int](func(a, b int) bool { return a < b })
	_, _, ok := h.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, h.Len())
}

func TestMinHeap_Remove(t *testing.T) {
	h := NewMinHeap[int, string](func(a, b int) bool { return a < b })
	h.Push(3, "three")
	h.Push(1, "one")
	h.Push(2, "two")

	removed := h.RemoveByValue("two")
	assert.True(t, removed)
	assert.Equal(t, 2, h.Len())

	key, val, _ := h.Pop()
	assert.Equal(t, 1, key)
	assert.Equal(t, "one", val)

	key, val, _ = h.Pop()
	assert.Equal(t, 3, key)
	assert.Equal(t, "three", val)
}
