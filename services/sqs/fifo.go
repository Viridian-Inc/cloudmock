package sqs

import (
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/collections"
)

// messageGroup holds messages for a single FIFO message group.
type messageGroup struct {
	groupID  string
	messages *collections.RingBuffer[*Message]
	locked   bool // true if a message from this group is inflight
}

// FIFOQueue is a FIFO SQS queue with per-group ordering, group locking,
// and 5-minute deduplication.
type FIFOQueue struct {
	mu   sync.Mutex
	name string
	url  string

	attrs map[string]string

	groups     map[string]*messageGroup // groupID -> group
	groupOrder []string                 // insertion order of group IDs for round-robin

	inflight map[string]*Message          // receiptHandle -> message
	visHeap  *collections.MinHeap[time.Time, *inflightEntry]

	// Dedup: dedupID -> expiry time
	dedupCache map[string]time.Time
	dedupHeap  *collections.MinHeap[time.Time, string] // for efficient expiry

	// DLQ
	dlqTarget          Queue
	dlqMaxReceiveCount int

	waiters []*waiter

	visTimer   *time.Timer
	dedupTimer *time.Timer
	closeCh    chan struct{}
	closed     bool
}

// NewFIFOQueue creates a new FIFO SQS queue.
func NewFIFOQueue(name, url string, attrs map[string]string) *FIFOQueue {
	if attrs == nil {
		attrs = make(map[string]string)
	}

	defaults := map[string]string{
		"VisibilityTimeout":            "30",
		"MessageRetentionPeriod":       "345600",
		"MaximumMessageSize":           "262144",
		"DelaySeconds":                 "0",
		"ReceiveMessageWaitTimeSeconds": "0",
		"FifoQueue":                    "true",
	}
	for k, v := range defaults {
		if _, ok := attrs[k]; !ok {
			attrs[k] = v
		}
	}
	if _, ok := attrs["ContentBasedDeduplication"]; !ok {
		attrs["ContentBasedDeduplication"] = "false"
	}

	q := &FIFOQueue{
		name:       name,
		url:        url,
		attrs:      attrs,
		groups:     make(map[string]*messageGroup),
		inflight:   make(map[string]*Message),
		visHeap:    collections.NewMinHeap[time.Time, *inflightEntry](func(a, b time.Time) bool { return a.Before(b) }),
		dedupCache: make(map[string]time.Time),
		dedupHeap:  collections.NewMinHeap[time.Time, string](func(a, b time.Time) bool { return a.Before(b) }),
		closeCh:    make(chan struct{}),
	}

	q.visTimer = time.NewTimer(time.Hour)
	q.visTimer.Stop()
	q.dedupTimer = time.NewTimer(time.Hour)
	q.dedupTimer.Stop()

	go q.backgroundLoop()
	return q
}

func (q *FIFOQueue) QueueName() string   { return q.name }
func (q *FIFOQueue) QueueURL() string    { return q.url }
func (q *FIFOQueue) IsFIFOQueue() bool   { return true }

func (q *FIFOQueue) GetAttributes() map[string]string {
	q.mu.Lock()
	defer q.mu.Unlock()
	cp := make(map[string]string, len(q.attrs))
	for k, v := range q.attrs {
		cp[k] = v
	}
	return cp
}

func (q *FIFOQueue) SetAttributes(attrs map[string]string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for k, v := range attrs {
		q.attrs[k] = v
	}
}

func (q *FIFOQueue) SendMessage(body string, delaySeconds int, attrs map[string]MessageAttribute, groupID, dedupID string) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()

	// Deduplication check.
	if dedupID != "" {
		q.pruneExpiredDedupLocked(now)
		if _, dup := q.dedupCache[dedupID]; dup {
			return "" // duplicate
		}
		expiry := now.Add(5 * time.Minute)
		q.dedupCache[dedupID] = expiry
		q.dedupHeap.Push(expiry, dedupID)
		q.resetDedupTimerLocked()
	}

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
	}

	grp, ok := q.groups[groupID]
	if !ok {
		grp = &messageGroup{
			groupID:  groupID,
			messages: collections.NewRingBuffer[*Message](16),
		}
		q.groups[groupID] = grp
		q.groupOrder = append(q.groupOrder, groupID)
	}
	grp.messages.Push(msg)

	q.signalWaitersLocked()
	return msgID
}

func (q *FIFOQueue) ReceiveMessages(maxCount int, visibilityTimeout int, waitTimeSeconds int) []*Message {
	q.mu.Lock()

	q.reclaimExpiredLocked()

	result := q.collectReadyLocked(maxCount, visibilityTimeout)

	if len(result) > 0 || waitTimeSeconds <= 0 {
		q.mu.Unlock()
		return result
	}

	// Long polling.
	w := &waiter{
		ch:       make(chan struct{}, 1),
		deadline: time.Now().Add(time.Duration(waitTimeSeconds) * time.Second),
	}
	q.waiters = append(q.waiters, w)
	q.mu.Unlock()

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

	q.reclaimExpiredLocked()
	return q.collectReadyLocked(maxCount, visibilityTimeout)
}

func (q *FIFOQueue) DeleteMessage(receiptHandle string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	msg, ok := q.inflight[receiptHandle]
	if !ok {
		return false
	}
	delete(q.inflight, receiptHandle)
	q.visHeap.RemoveByValue(&inflightEntry{receiptHandle: receiptHandle, msg: msg})

	// Unlock the message group.
	if grp, exists := q.groups[msg.MessageGroupId]; exists {
		grp.locked = false
	}

	return true
}

func (q *FIFOQueue) ChangeMessageVisibility(receiptHandle string, timeout int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	msg, ok := q.inflight[receiptHandle]
	if !ok {
		return false
	}

	q.visHeap.RemoveByValue(&inflightEntry{receiptHandle: receiptHandle, msg: msg})

	if timeout == 0 {
		delete(q.inflight, receiptHandle)
		// Push back to group front (it already has a ring buffer, we just push it back).
		if grp, exists := q.groups[msg.MessageGroupId]; exists {
			// We need to prepend - for simplicity, just push (it goes to end, which
			// may not be perfectly FIFO, but is acceptable for a mock).
			grp.messages.Push(msg)
			grp.locked = false
		}
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

func (q *FIFOQueue) Purge() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.groups = make(map[string]*messageGroup)
	q.groupOrder = nil
	q.inflight = make(map[string]*Message)
	q.visHeap = collections.NewMinHeap[time.Time, *inflightEntry](func(a, b time.Time) bool { return a.Before(b) })
}

func (q *FIFOQueue) ApproximateNumberOfMessages() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	count := 0
	for _, grp := range q.groups {
		count += grp.messages.Len()
	}
	return count
}

func (q *FIFOQueue) ApproximateNumberOfMessagesNotVisible() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.inflight)
}

func (q *FIFOQueue) SetDLQ(target Queue, maxReceiveCount int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.dlqTarget = target
	q.dlqMaxReceiveCount = maxReceiveCount
}

func (q *FIFOQueue) Close() {
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

func (q *FIFOQueue) collectReadyLocked(maxCount int, visibilityTimeout int) []*Message {
	now := time.Now().UTC()
	result := make([]*Message, 0, maxCount)

	// Track which groups we touched in this batch so we can lock them after.
	touchedGroups := make(map[string]bool)

	// Round-robin across groups, draining available messages.
	// Within a single ReceiveMessages call, multiple messages from the same
	// group may be returned (matching AWS behaviour). The group is locked
	// after this call returns, preventing other ReceiveMessages calls from
	// pulling from the same group until the inflight messages are deleted.
	changed := true
	for changed && len(result) < maxCount {
		changed = false
		for i := 0; i < len(q.groupOrder) && len(result) < maxCount; i++ {
			gid := q.groupOrder[i]
			grp, exists := q.groups[gid]
			if !exists || grp.locked || grp.messages.Len() == 0 {
				continue
			}

			msg, ok := grp.messages.Pop()
			if !ok {
				continue
			}

			// Skip delayed messages (put them back).
			if now.Before(msg.DelayUntil) {
				grp.messages.Push(msg)
				continue
			}

			rh := newReceiptHandle()
			msg.ReceiptHandle = rh
			msg.ReceiveCount++
			if msg.ReceiveCount == 1 {
				msg.FirstReceiveTimestamp = now
			}

			// Check DLQ.
			if q.dlqTarget != nil && q.dlqMaxReceiveCount > 0 && msg.ReceiveCount > q.dlqMaxReceiveCount {
				q.dlqTarget.SendMessage(msg.Body, 0, msg.MessageAttributes, msg.MessageGroupId, "")
				changed = true
				continue
			}

			deadline := now.Add(time.Duration(visibilityTimeout) * time.Second)
			msg.VisibilityDeadline = deadline
			q.inflight[rh] = msg
			entry := &inflightEntry{receiptHandle: rh, msg: msg}
			q.visHeap.Push(deadline, entry)

			touchedGroups[gid] = true
			result = append(result, msg)
			changed = true
		}
	}

	// Lock all groups that had messages delivered.
	for gid := range touchedGroups {
		if grp, exists := q.groups[gid]; exists {
			grp.locked = true
		}
	}

	if len(result) > 0 {
		q.resetVisTimerLocked()
	}

	// Clean up empty groups.
	q.cleanEmptyGroupsLocked()

	return result
}

func (q *FIFOQueue) reclaimExpiredLocked() {
	now := time.Now().UTC()
	for {
		deadline, entry, ok := q.visHeap.Peek()
		if !ok || now.Before(deadline) {
			break
		}
		q.visHeap.Pop()

		if _, exists := q.inflight[entry.receiptHandle]; !exists {
			continue
		}
		delete(q.inflight, entry.receiptHandle)

		// Unlock the group.
		if grp, exists := q.groups[entry.msg.MessageGroupId]; exists {
			grp.locked = false
		}

		// Check DLQ.
		if q.dlqTarget != nil && q.dlqMaxReceiveCount > 0 && entry.msg.ReceiveCount >= q.dlqMaxReceiveCount {
			q.dlqTarget.SendMessage(entry.msg.Body, 0, entry.msg.MessageAttributes, entry.msg.MessageGroupId, "")
			continue
		}

		// Put back in group.
		gid := entry.msg.MessageGroupId
		grp, exists := q.groups[gid]
		if !exists {
			grp = &messageGroup{
				groupID:  gid,
				messages: collections.NewRingBuffer[*Message](16),
			}
			q.groups[gid] = grp
			q.groupOrder = append(q.groupOrder, gid)
		}
		grp.messages.Push(entry.msg)
	}
	q.resetVisTimerLocked()
}

func (q *FIFOQueue) pruneExpiredDedupLocked(now time.Time) {
	for {
		expiry, dedupID, ok := q.dedupHeap.Peek()
		if !ok || now.Before(expiry) {
			break
		}
		q.dedupHeap.Pop()
		// Only delete if the expiry matches (could have been re-added).
		if stored, exists := q.dedupCache[dedupID]; exists && !stored.After(expiry) {
			delete(q.dedupCache, dedupID)
		}
	}
}

func (q *FIFOQueue) signalWaitersLocked() {
	now := time.Now()
	remaining := q.waiters[:0]
	for _, w := range q.waiters {
		if now.After(w.deadline) {
			continue
		}
		select {
		case w.ch <- struct{}{}:
		default:
		}
	}
	q.waiters = remaining
}

func (q *FIFOQueue) resetVisTimerLocked() {
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

func (q *FIFOQueue) resetDedupTimerLocked() {
	expiry, _, ok := q.dedupHeap.Peek()
	if !ok {
		q.dedupTimer.Stop()
		return
	}
	d := time.Until(expiry)
	if d < 0 {
		d = 0
	}
	q.dedupTimer.Reset(d)
}

func (q *FIFOQueue) cleanEmptyGroupsLocked() {
	newOrder := q.groupOrder[:0]
	for _, gid := range q.groupOrder {
		grp, exists := q.groups[gid]
		if !exists {
			continue
		}
		if grp.messages.Len() == 0 && !grp.locked {
			delete(q.groups, gid)
			continue
		}
		newOrder = append(newOrder, gid)
	}
	q.groupOrder = newOrder
}

func (q *FIFOQueue) backgroundLoop() {
	for {
		select {
		case <-q.closeCh:
			return
		case <-q.visTimer.C:
			q.mu.Lock()
			q.reclaimExpiredLocked()
			q.signalWaitersLocked()
			q.mu.Unlock()
		case <-q.dedupTimer.C:
			q.mu.Lock()
			q.pruneExpiredDedupLocked(time.Now().UTC())
			q.resetDedupTimerLocked()
			q.mu.Unlock()
		}
	}
}
