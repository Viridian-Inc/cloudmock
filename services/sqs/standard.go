package sqs

import (
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/collections"
)

// inflightEntry tracks an inflight message and its visibility deadline.
type inflightEntry struct {
	receiptHandle string
	msg           *Message
}

// waiter represents a goroutine blocked in a long-poll ReceiveMessages call.
type waiter struct {
	ch       chan struct{}
	deadline time.Time
}

// StandardQueue is a standard (non-FIFO) SQS queue using a ring buffer for
// ready messages, min-heaps for delayed and visibility-timeout tracking,
// and channel-based long polling.
type StandardQueue struct {
	mu   sync.Mutex
	name string
	url  string

	attrs map[string]string

	ready    *collections.RingBuffer[*Message]
	delayed  *collections.MinHeap[time.Time, *Message]
	inflight map[string]*Message // keyed by ReceiptHandle
	visHeap  *collections.MinHeap[time.Time, *inflightEntry]

	waiters []*waiter

	// DLQ settings
	dlqTarget          Queue
	dlqMaxReceiveCount int

	delayTimer *time.Timer
	visTimer   *time.Timer
	closeCh    chan struct{}
	closed     bool
}

// NewStandardQueue creates a new standard SQS queue.
func NewStandardQueue(name, url string, attrs map[string]string) *StandardQueue {
	if attrs == nil {
		attrs = make(map[string]string)
	}

	// Set defaults.
	defaults := map[string]string{
		"VisibilityTimeout":             "30",
		"MessageRetentionPeriod":        "345600",
		"MaximumMessageSize":            "262144",
		"DelaySeconds":                  "0",
		"ReceiveMessageWaitTimeSeconds": "0",
	}
	for k, v := range defaults {
		if _, ok := attrs[k]; !ok {
			attrs[k] = v
		}
	}

	q := &StandardQueue{
		name:     name,
		url:      url,
		attrs:    attrs,
		ready:    collections.NewRingBuffer[*Message](64),
		delayed:  collections.NewMinHeap[time.Time, *Message](func(a, b time.Time) bool { return a.Before(b) }),
		inflight: make(map[string]*Message),
		visHeap:  collections.NewMinHeap[time.Time, *inflightEntry](func(a, b time.Time) bool { return a.Before(b) }),
		closeCh:  make(chan struct{}),
	}

	// Start background timers.
	q.delayTimer = time.NewTimer(time.Hour)
	q.delayTimer.Stop()
	q.visTimer = time.NewTimer(time.Hour)
	q.visTimer.Stop()

	go q.backgroundLoop()
	return q
}

func (q *StandardQueue) QueueName() string { return q.name }
func (q *StandardQueue) QueueURL() string  { return q.url }
func (q *StandardQueue) IsFIFOQueue() bool { return false }

func (q *StandardQueue) GetAttributes() map[string]string {
	q.mu.Lock()
	defer q.mu.Unlock()
	cp := make(map[string]string, len(q.attrs))
	for k, v := range q.attrs {
		cp[k] = v
	}
	return cp
}

func (q *StandardQueue) SetAttributes(attrs map[string]string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for k, v := range attrs {
		q.attrs[k] = v
	}
}

func (q *StandardQueue) SendMessage(body string, delaySeconds int, attrs map[string]MessageAttribute, groupID, dedupID string) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	msgID := newUUID()

	msg := &Message{
		MessageId:              msgID,
		Body:                   body,
		MD5OfBody:              md5Hex(body),
		SentTimestamp:          now,
		MessageAttributes:      attrs,
		MessageGroupId:         groupID,
		MessageDeduplicationId: dedupID,
	}

	if delaySeconds > 0 {
		msg.DelayUntil = now.Add(time.Duration(delaySeconds) * time.Second)
		q.delayed.Push(msg.DelayUntil, msg)
		q.resetDelayTimerLocked()
	} else {
		q.ready.Push(msg)
		q.signalWaitersLocked()
	}

	return msgID
}

func (q *StandardQueue) ReceiveMessages(maxCount int, visibilityTimeout int, waitTimeSeconds int) []*Message {
	q.mu.Lock()

	// First, mature any delayed messages and reclaim expired inflight.
	q.matureDelayedLocked()
	q.reclaimExpiredLocked()

	result := q.collectReadyLocked(maxCount, visibilityTimeout)

	if len(result) > 0 || waitTimeSeconds <= 0 {
		q.mu.Unlock()
		return result
	}

	// Long polling: register a waiter and wait.
	w := &waiter{
		ch:       make(chan struct{}, 1),
		deadline: time.Now().Add(time.Duration(waitTimeSeconds) * time.Second),
	}
	q.waiters = append(q.waiters, w)
	q.mu.Unlock()

	// Block until signaled or timeout.
	timer := time.NewTimer(time.Duration(waitTimeSeconds) * time.Second)
	defer timer.Stop()

	select {
	case <-w.ch:
	case <-timer.C:
	case <-q.closeCh:
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.matureDelayedLocked()
	q.reclaimExpiredLocked()

	return q.collectReadyLocked(maxCount, visibilityTimeout)
}

func (q *StandardQueue) DeleteMessage(receiptHandle string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	msg, ok := q.inflight[receiptHandle]
	if !ok {
		return false
	}
	delete(q.inflight, receiptHandle)
	q.visHeap.RemoveByValue(&inflightEntry{receiptHandle: receiptHandle, msg: msg})
	return true
}

func (q *StandardQueue) ChangeMessageVisibility(receiptHandle string, timeout int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	msg, ok := q.inflight[receiptHandle]
	if !ok {
		return false
	}

	// Remove old entry from vis heap.
	q.visHeap.RemoveByValue(&inflightEntry{receiptHandle: receiptHandle, msg: msg})

	if timeout == 0 {
		// Make immediately visible: remove from inflight, push to ready.
		delete(q.inflight, receiptHandle)
		q.ready.Push(msg)
		q.signalWaitersLocked()
	} else {
		newDeadline := time.Now().UTC().Add(time.Duration(timeout) * time.Second)
		msg.VisibilityDeadline = newDeadline
		entry := &inflightEntry{receiptHandle: receiptHandle, msg: msg}
		q.visHeap.Push(newDeadline, entry)
		q.resetVisTimerLocked()
	}

	return true
}

func (q *StandardQueue) Purge() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.ready = collections.NewRingBuffer[*Message](64)
	q.delayed = collections.NewMinHeap[time.Time, *Message](func(a, b time.Time) bool { return a.Before(b) })
	q.inflight = make(map[string]*Message)
	q.visHeap = collections.NewMinHeap[time.Time, *inflightEntry](func(a, b time.Time) bool { return a.Before(b) })
}

func (q *StandardQueue) ApproximateNumberOfMessages() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.ready.Len() + q.delayed.Len()
}

func (q *StandardQueue) ApproximateNumberOfMessagesNotVisible() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.inflight)
}

func (q *StandardQueue) SetDLQ(target Queue, maxReceiveCount int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.dlqTarget = target
	q.dlqMaxReceiveCount = maxReceiveCount
}

func (q *StandardQueue) Close() {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.closed = true
	close(q.closeCh)
	q.mu.Unlock()
}

// ---- internal helpers (caller must hold q.mu) ----

func (q *StandardQueue) collectReadyLocked(maxCount int, visibilityTimeout int) []*Message {
	now := time.Now().UTC()
	result := make([]*Message, 0, maxCount)

	for len(result) < maxCount {
		msg, ok := q.ready.Pop()
		if !ok {
			break
		}

		rh := newReceiptHandle()
		msg.ReceiptHandle = rh
		msg.ReceiveCount++
		if msg.ReceiveCount == 1 {
			msg.FirstReceiveTimestamp = now
		}

		// Check DLQ threshold before delivering.
		if q.dlqTarget != nil && q.dlqMaxReceiveCount > 0 && msg.ReceiveCount > q.dlqMaxReceiveCount {
			q.dlqTarget.SendMessage(msg.Body, 0, msg.MessageAttributes, msg.MessageGroupId, "")
			continue
		}

		deadline := now.Add(time.Duration(visibilityTimeout) * time.Second)
		msg.VisibilityDeadline = deadline
		q.inflight[rh] = msg

		entry := &inflightEntry{receiptHandle: rh, msg: msg}
		q.visHeap.Push(deadline, entry)

		result = append(result, msg)
	}

	if len(result) > 0 {
		q.resetVisTimerLocked()
	}

	return result
}

func (q *StandardQueue) matureDelayedLocked() {
	now := time.Now().UTC()
	for {
		deadline, msg, ok := q.delayed.Peek()
		if !ok || now.Before(deadline) {
			break
		}
		q.delayed.Pop()
		_ = msg
		q.ready.Push(msg)
	}
	q.resetDelayTimerLocked()
}

func (q *StandardQueue) reclaimExpiredLocked() {
	now := time.Now().UTC()
	for {
		deadline, entry, ok := q.visHeap.Peek()
		if !ok || now.Before(deadline) {
			break
		}
		q.visHeap.Pop()

		// Verify still inflight (could have been deleted).
		if _, exists := q.inflight[entry.receiptHandle]; !exists {
			continue
		}
		delete(q.inflight, entry.receiptHandle)

		// Check DLQ threshold.
		if q.dlqTarget != nil && q.dlqMaxReceiveCount > 0 && entry.msg.ReceiveCount >= q.dlqMaxReceiveCount {
			q.dlqTarget.SendMessage(entry.msg.Body, 0, entry.msg.MessageAttributes, entry.msg.MessageGroupId, "")
			continue
		}

		q.ready.Push(entry.msg)
	}
	q.resetVisTimerLocked()
}

func (q *StandardQueue) signalWaitersLocked() {
	now := time.Now()
	remaining := q.waiters[:0]
	for _, w := range q.waiters {
		if now.After(w.deadline) {
			continue // expired waiter, skip
		}
		select {
		case w.ch <- struct{}{}:
		default:
		}
	}
	q.waiters = remaining
}

func (q *StandardQueue) resetDelayTimerLocked() {
	deadline, _, ok := q.delayed.Peek()
	if !ok {
		q.delayTimer.Stop()
		return
	}
	d := time.Until(deadline)
	if d < 0 {
		d = 0
	}
	q.delayTimer.Reset(d)
}

func (q *StandardQueue) resetVisTimerLocked() {
	deadline, _, ok := q.visHeap.Peek()
	if !ok {
		q.visTimer.Stop()
		return
	}
	d := time.Until(deadline)
	if d < 0 {
		d = 0
	}
	q.visTimer.Reset(d)
}

func (q *StandardQueue) backgroundLoop() {
	for {
		select {
		case <-q.closeCh:
			return
		case <-q.delayTimer.C:
			q.mu.Lock()
			q.matureDelayedLocked()
			if q.ready.Len() > 0 {
				q.signalWaitersLocked()
			}
			q.mu.Unlock()
		case <-q.visTimer.C:
			q.mu.Lock()
			q.reclaimExpiredLocked()
			if q.ready.Len() > 0 {
				q.signalWaitersLocked()
			}
			q.mu.Unlock()
		}
	}
}

// newStandardQueue is a helper used by the store. It mirrors the old newQueue defaults.
func newStandardQueue(name, url string, attrs map[string]string) *StandardQueue {
	if attrs == nil {
		attrs = make(map[string]string)
	}
	if strings.HasSuffix(name, ".fifo") {
		// Should not happen — store routes .fifo to NewFIFOQueue.
		// But handle gracefully.
		attrs["FifoQueue"] = "true"
	}
	return NewStandardQueue(name, url, attrs)
}
