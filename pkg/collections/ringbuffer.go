package collections

type RingBuffer[T any] struct {
	buf  []T
	head int
	tail int
	len  int
	cap  int
}

func NewRingBuffer[T any](initialCap int) *RingBuffer[T] {
	if initialCap < 4 {
		initialCap = 4
	}
	return &RingBuffer[T]{
		buf: make([]T, initialCap),
		cap: initialCap,
	}
}

func (rb *RingBuffer[T]) Push(item T) {
	if rb.len == rb.cap {
		rb.grow()
	}
	rb.buf[rb.tail] = item
	rb.tail = (rb.tail + 1) % rb.cap
	rb.len++
}

func (rb *RingBuffer[T]) Pop() (T, bool) {
	if rb.len == 0 {
		var zero T
		return zero, false
	}
	item := rb.buf[rb.head]
	var zero T
	rb.buf[rb.head] = zero
	rb.head = (rb.head + 1) % rb.cap
	rb.len--
	return item, true
}

func (rb *RingBuffer[T]) Len() int { return rb.len }

func (rb *RingBuffer[T]) grow() {
	newCap := rb.cap * 2
	newBuf := make([]T, newCap)
	if rb.head < rb.tail {
		copy(newBuf, rb.buf[rb.head:rb.tail])
	} else {
		n := copy(newBuf, rb.buf[rb.head:])
		copy(newBuf[n:], rb.buf[:rb.tail])
	}
	rb.buf = newBuf
	rb.head = 0
	rb.tail = rb.len
	rb.cap = newCap
}
