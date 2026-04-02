package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBuffer_PushPop(t *testing.T) {
	rb := NewRingBuffer[int](4)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	val, ok := rb.Pop()
	require.True(t, ok)
	assert.Equal(t, 1, val)
	val, ok = rb.Pop()
	require.True(t, ok)
	assert.Equal(t, 2, val)
	assert.Equal(t, 1, rb.Len())
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer[string](4)
	_, ok := rb.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, rb.Len())
}

func TestRingBuffer_Grow(t *testing.T) {
	rb := NewRingBuffer[int](2)
	for i := 0; i < 100; i++ {
		rb.Push(i)
	}
	assert.Equal(t, 100, rb.Len())
	for i := 0; i < 100; i++ {
		val, ok := rb.Pop()
		require.True(t, ok)
		assert.Equal(t, i, val)
	}
	assert.Equal(t, 0, rb.Len())
}

func TestRingBuffer_Wraparound(t *testing.T) {
	rb := NewRingBuffer[int](4)
	rb.Push(1)
	rb.Push(2)
	rb.Pop()
	rb.Pop()
	rb.Push(3)
	rb.Push(4)
	rb.Push(5)
	rb.Push(6)

	assert.Equal(t, 4, rb.Len())
	val, _ := rb.Pop()
	assert.Equal(t, 3, val)
	val, _ = rb.Pop()
	assert.Equal(t, 4, val)
	val, _ = rb.Pop()
	assert.Equal(t, 5, val)
	val, _ = rb.Pop()
	assert.Equal(t, 6, val)
}

func BenchmarkRingBuffer_PushPop(b *testing.B) {
	rb := NewRingBuffer[int](1024)
	for i := 0; i < b.N; i++ {
		rb.Push(i)
		rb.Pop()
	}
}
