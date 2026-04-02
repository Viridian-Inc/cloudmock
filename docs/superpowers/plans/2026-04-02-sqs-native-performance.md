# SQS Native Performance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite SQS queue internals to achieve <0.05ms P50 send/receive, lock-free ring buffer, timer-driven visibility/delay, long polling, FIFO with message groups, and dead-letter queues.

**Architecture:** Replace slice-based queue with ring buffer for ready messages, min-heaps for delayed/inflight timers, channel-based long polling, per-group FIFO ordering, and DLQ redrive.

**Tech Stack:** Go 1.26, `pkg/collections` (ring buffer, min-heap from DynamoDB plan), `time.Timer` for async expiry

**Depends on:** Plan A (DynamoDB) Task 1 (min-heap) and Task 2 (ring buffer) must be completed first.

---

## File Structure

```
services/sqs/
├── service.go         # KEEP — minimal wiring changes
├── handlers.go        # MODIFY — add long poll support, DLQ attributes
├── json_handlers.go   # KEEP — unchanged
├── store.go           # REWRITE — add ARN index, DLQ resolution
├── queue.go           # DELETE — replaced by standard.go + fifo.go
├── standard.go        # NEW — ring buffer queue with timers
├── fifo.go            # NEW — FIFO queue with message groups
├── longpoll.go        # NEW — waiter management
├── dlq.go             # NEW — dead-letter queue redrive
├── message.go         # NEW — Message type + helpers (extracted from queue.go)
├── standard_test.go   # NEW — standard queue tests
├── fifo_test.go       # NEW — FIFO queue tests
├── longpoll_test.go   # NEW — long polling tests
├── dlq_test.go        # NEW — DLQ tests
├── bench_test.go      # NEW — Go benchmarks
├── service_test.go    # KEEP — existing tests must pass
└── json_test.go       # KEEP — existing tests must pass
```

---

### Task 1: Message Type Extraction

**Files:**
- Create: `services/sqs/message.go`

- [ ] **Step 1: Create message.go**

Extract `Message`, `MessageAttribute`, and helper functions from `queue.go` into their own file:

```go
package sqs

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageAttribute holds a typed message attribute.
type MessageAttribute struct {
	DataType    string
	StringValue string
	BinaryValue []byte
}

// Message represents an SQS message.
type Message struct {
	MessageId                string
	Body                     string
	ReceiptHandle            string
	MD5OfBody                string
	SentTimestamp            time.Time
	FirstReceiveTimestamp    time.Time
	ReceiveCount             int
	VisibilityDeadline       time.Time
	DelayUntil               time.Time
	MessageAttributes        map[string]MessageAttribute
	MessageGroupId           string
	MessageDeduplicationId   string
}

// Queue is the interface implemented by both StandardQueue and FIFOQueue.
type Queue interface {
	QueueName() string
	QueueURL() string
	QueueARN() string
	IsFIFOQueue() bool
	Attributes() map[string]string
	SetAttributes(attrs map[string]string)

	SendMessage(body string, delaySeconds int, attrs map[string]MessageAttribute, groupID, dedupID string) (string, error)
	ReceiveMessages(maxCount int, visibilityTimeout int, waitTimeSeconds int) []*Message
	DeleteMessage(receiptHandle string) bool
	ChangeMessageVisibility(receiptHandle string, timeout int) bool
	Purge()
	ApproximateNumberOfMessages() int
	ApproximateNumberOfMessagesNotVisible() int

	SetDLQ(target Queue, maxReceiveCount int)
	Close() // stop timers
}

func md5Hex(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func newMessageID() string {
	return uuid.New().String()
}

func newReceiptHandle() string {
	return uuid.New().String()
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/megan/cloudmock && go build ./services/sqs/`
Expected: May fail because queue.go still exists — that's OK, we'll delete it in the next task.

- [ ] **Step 3: Commit**

```bash
git add services/sqs/message.go
git commit -m "feat(sqs): extract message types to dedicated file"
```

---

### Task 2: Standard Queue with Ring Buffer

**Files:**
- Create: `services/sqs/standard.go`
- Create: `services/sqs/standard_test.go`
- Delete: `services/sqs/queue.go` (after standard.go compiles)

- [ ] **Step 1: Write the failing test**

Create `services/sqs/standard_test.go`:

```go
package sqs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardQueue_SendReceiveDelete(t *testing.T) {
	q := NewStandardQueue("test-queue", "http://localhost/queue/test-queue", "arn:aws:sqs:us-east-1:000:test-queue")
	defer q.Close()

	msgID, err := q.SendMessage("hello", 0, nil, "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, msgID)

	msgs := q.ReceiveMessages(1, 30, 0)
	require.Len(t, msgs, 1)
	assert.Equal(t, "hello", msgs[0].Body)
	assert.Equal(t, 1, msgs[0].ReceiveCount)

	ok := q.DeleteMessage(msgs[0].ReceiptHandle)
	assert.True(t, ok)
	assert.Equal(t, 0, q.ApproximateNumberOfMessages())
}

func TestStandardQueue_VisibilityTimeout(t *testing.T) {
	q := NewStandardQueue("vis-test", "http://localhost/queue/vis-test", "arn:aws:sqs:us-east-1:000:vis-test")
	defer q.Close()

	q.SendMessage("reappear", 0, nil, "", "")

	msgs := q.ReceiveMessages(1, 1, 0) // 1 second visibility
	require.Len(t, msgs, 1)

	// Immediately: message should be inflight, not available
	empty := q.ReceiveMessages(1, 30, 0)
	assert.Len(t, empty, 0)

	// Wait for visibility timeout
	time.Sleep(1500 * time.Millisecond)

	// Message should reappear
	msgs2 := q.ReceiveMessages(1, 30, 0)
	require.Len(t, msgs2, 1)
	assert.Equal(t, "reappear", msgs2[0].Body)
	assert.Equal(t, 2, msgs2[0].ReceiveCount)
}

func TestStandardQueue_DelayedMessage(t *testing.T) {
	q := NewStandardQueue("delay-test", "http://localhost/queue/delay-test", "arn:aws:sqs:us-east-1:000:delay-test")
	defer q.Close()

	q.SendMessage("delayed", 1, nil, "", "")

	// Not available immediately
	msgs := q.ReceiveMessages(1, 30, 0)
	assert.Len(t, msgs, 0)

	// Available after delay
	time.Sleep(1200 * time.Millisecond)
	msgs = q.ReceiveMessages(1, 30, 0)
	require.Len(t, msgs, 1)
	assert.Equal(t, "delayed", msgs[0].Body)
}

func TestStandardQueue_Purge(t *testing.T) {
	q := NewStandardQueue("purge-test", "http://localhost/queue/purge-test", "arn:aws:sqs:us-east-1:000:purge-test")
	defer q.Close()

	for i := 0; i < 100; i++ {
		q.SendMessage("msg", 0, nil, "", "")
	}
	assert.Equal(t, 100, q.ApproximateNumberOfMessages())

	q.Purge()
	assert.Equal(t, 0, q.ApproximateNumberOfMessages())
}

func TestStandardQueue_BatchSendReceive(t *testing.T) {
	q := NewStandardQueue("batch-test", "http://localhost/queue/batch-test", "arn:aws:sqs:us-east-1:000:batch-test")
	defer q.Close()

	for i := 0; i < 20; i++ {
		q.SendMessage("msg", 0, nil, "", "")
	}

	msgs := q.ReceiveMessages(10, 30, 0)
	assert.Equal(t, 10, len(msgs))

	msgs2 := q.ReceiveMessages(10, 30, 0)
	assert.Equal(t, 10, len(msgs2))
}

func BenchmarkStandardQueue_SendReceive(b *testing.B) {
	q := NewStandardQueue("bench", "http://localhost/queue/bench", "arn:aws:sqs:us-east-1:000:bench")
	defer q.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.SendMessage("bench-msg", 0, nil, "", "")
		q.ReceiveMessages(1, 30, 0)
	}
}
```

- [ ] **Step 2: Implement standard.go**

Create `services/sqs/standard.go` implementing the `Queue` interface with:
- `RingBuffer[*Message]` for ready messages
- `MinHeap[time.Time, *Message]` for delayed messages with timer
- `map[string]*Message` for inflight + `MinHeap` for visibility timeouts with timer
- Timer-driven maturation of delayed messages and visibility expiry

Key methods:
- `NewStandardQueue(name, url, arn string) *StandardQueue`
- `SendMessage`: push to ready ring buffer (or delayed heap if delay > 0), wake waiters
- `ReceiveMessages`: pop from ready, set visibility deadline, push to inflight
- `DeleteMessage`: remove from inflight map
- `ChangeMessageVisibility`: update deadline in inflight
- `Close`: stop all timers

- [ ] **Step 3: Delete old queue.go**

Remove `services/sqs/queue.go` — all its functionality is now in `message.go` + `standard.go`.

- [ ] **Step 4: Run tests**

Run: `cd /Users/megan/cloudmock && go test ./services/sqs/ -run TestStandard -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git rm services/sqs/queue.go
git add services/sqs/standard.go services/sqs/standard_test.go
git commit -m "feat(sqs): standard queue with ring buffer and timer-driven visibility"
```

---

### Task 3: Long Polling

**Files:**
- Create: `services/sqs/longpoll.go`
- Create: `services/sqs/longpoll_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/sqs/longpoll_test.go`:

```go
package sqs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLongPoll_MessageArrivesWhileWaiting(t *testing.T) {
	q := NewStandardQueue("poll-test", "http://localhost/queue/poll-test", "arn:aws:sqs:us-east-1:000:poll-test")
	defer q.Close()

	done := make(chan []*Message, 1)
	go func() {
		msgs := q.ReceiveMessages(1, 30, 5) // 5 second wait
		done <- msgs
	}()

	time.Sleep(100 * time.Millisecond) // let the goroutine start waiting
	q.SendMessage("arrived!", 0, nil, "", "")

	select {
	case msgs := <-done:
		require.Len(t, msgs, 1)
		assert.Equal(t, "arrived!", msgs[0].Body)
	case <-time.After(2 * time.Second):
		t.Fatal("long poll did not return within 2s")
	}
}

func TestLongPoll_TimeoutReturnsEmpty(t *testing.T) {
	q := NewStandardQueue("poll-timeout", "http://localhost/queue/poll-timeout", "arn:aws:sqs:us-east-1:000:poll-timeout")
	defer q.Close()

	start := time.Now()
	msgs := q.ReceiveMessages(1, 30, 1) // 1 second wait
	elapsed := time.Since(start)

	assert.Len(t, msgs, 0)
	assert.Greater(t, elapsed, 900*time.Millisecond)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestLongPoll_ImmediateReturnWhenMessagesExist(t *testing.T) {
	q := NewStandardQueue("poll-immediate", "http://localhost/queue/poll-immediate", "arn:aws:sqs:us-east-1:000:poll-immediate")
	defer q.Close()

	q.SendMessage("existing", 0, nil, "", "")

	start := time.Now()
	msgs := q.ReceiveMessages(1, 30, 20) // 20s wait, but should return immediately
	elapsed := time.Since(start)

	require.Len(t, msgs, 1)
	assert.Less(t, elapsed, 100*time.Millisecond)
}
```

- [ ] **Step 2: Implement longpoll.go**

Create `services/sqs/longpoll.go`:

```go
package sqs

import "sync"

// waiterPool manages long-poll waiters for a queue.
type waiterPool struct {
	mu      sync.Mutex
	waiters []chan struct{} // signaled when messages arrive
}

func (wp *waiterPool) addWaiter() chan struct{} {
	ch := make(chan struct{}, 1)
	wp.mu.Lock()
	wp.waiters = append(wp.waiters, ch)
	wp.mu.Unlock()
	return ch
}

func (wp *waiterPool) removeWaiter(ch chan struct{}) {
	wp.mu.Lock()
	for i, w := range wp.waiters {
		if w == ch {
			wp.waiters = append(wp.waiters[:i], wp.waiters[i+1:]...)
			break
		}
	}
	wp.mu.Unlock()
}

// signal wakes one waiter (if any). Called when a message is enqueued.
func (wp *waiterPool) signal() {
	wp.mu.Lock()
	if len(wp.waiters) > 0 {
		ch := wp.waiters[0]
		wp.waiters = wp.waiters[1:]
		wp.mu.Unlock()
		select {
		case ch <- struct{}{}:
		default:
		}
		return
	}
	wp.mu.Unlock()
}
```

Then integrate into `StandardQueue.ReceiveMessages`:
- If queue empty and `waitTimeSeconds > 0`: register waiter, block on channel with timeout
- `SendMessage`: after pushing to ready, call `waiters.signal()`

- [ ] **Step 3: Run tests**

Run: `cd /Users/megan/cloudmock && go test ./services/sqs/ -run TestLongPoll -v -timeout 30s`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add services/sqs/longpoll.go services/sqs/longpoll_test.go
git commit -m "feat(sqs): long polling with channel-based waiters"
```

---

### Task 4: FIFO Queue

**Files:**
- Create: `services/sqs/fifo.go`
- Create: `services/sqs/fifo_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/sqs/fifo_test.go`:

```go
package sqs

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFIFOQueue_OrderPreserved(t *testing.T) {
	q := NewFIFOQueue("test.fifo", "http://localhost/queue/test.fifo", "arn:aws:sqs:us-east-1:000:test.fifo")
	defer q.Close()

	for i := 0; i < 10; i++ {
		q.SendMessage(fmt.Sprintf("msg-%d", i), 0, nil, "group-1", fmt.Sprintf("dedup-%d", i))
	}

	for i := 0; i < 10; i++ {
		msgs := q.ReceiveMessages(1, 30, 0)
		require.Len(t, msgs, 1)
		assert.Equal(t, fmt.Sprintf("msg-%d", i), msgs[0].Body)
		q.DeleteMessage(msgs[0].ReceiptHandle)
	}
}

func TestFIFOQueue_GroupIsolation(t *testing.T) {
	q := NewFIFOQueue("groups.fifo", "http://localhost/queue/groups.fifo", "arn:aws:sqs:us-east-1:000:groups.fifo")
	defer q.Close()

	q.SendMessage("g1-msg1", 0, nil, "group-1", "d1")
	q.SendMessage("g2-msg1", 0, nil, "group-2", "d2")
	q.SendMessage("g1-msg2", 0, nil, "group-1", "d3")

	// First receive gets one from each group (group-1 is locked after first)
	msgs := q.ReceiveMessages(2, 30, 0)
	require.Len(t, msgs, 2)

	bodies := map[string]bool{msgs[0].Body: true, msgs[1].Body: true}
	assert.True(t, bodies["g1-msg1"])
	assert.True(t, bodies["g2-msg1"])

	// group-1's second message is blocked (group locked)
	msgs2 := q.ReceiveMessages(1, 30, 0)
	assert.Len(t, msgs2, 0)

	// Delete first group-1 message, second should become available
	for _, m := range msgs {
		if m.Body == "g1-msg1" {
			q.DeleteMessage(m.ReceiptHandle)
		}
	}

	msgs3 := q.ReceiveMessages(1, 30, 0)
	require.Len(t, msgs3, 1)
	assert.Equal(t, "g1-msg2", msgs3[0].Body)
}

func TestFIFOQueue_Deduplication(t *testing.T) {
	q := NewFIFOQueue("dedup.fifo", "http://localhost/queue/dedup.fifo", "arn:aws:sqs:us-east-1:000:dedup.fifo")
	defer q.Close()

	id1, _ := q.SendMessage("first", 0, nil, "g1", "same-dedup")
	id2, _ := q.SendMessage("duplicate", 0, nil, "g1", "same-dedup")

	assert.Equal(t, id1, id2) // same dedup ID returns same message ID
	assert.Equal(t, 1, q.ApproximateNumberOfMessages()) // only 1 message
}

func TestFIFOQueue_DedupExpires(t *testing.T) {
	q := NewFIFOQueue("dedup-exp.fifo", "http://localhost/queue/dedup-exp.fifo", "arn:aws:sqs:us-east-1:000:dedup-exp.fifo")
	defer q.Close()

	// Override dedup window for testing (normally 5 minutes)
	q.(*FIFOQueue).dedupWindow = 500 * time.Millisecond

	q.SendMessage("first", 0, nil, "g1", "dedup-1")
	time.Sleep(600 * time.Millisecond)
	q.SendMessage("second", 0, nil, "g1", "dedup-1")

	assert.Equal(t, 2, q.ApproximateNumberOfMessages()) // dedup expired, both accepted
}
```

- [ ] **Step 2: Implement fifo.go**

Create `services/sqs/fifo.go` implementing the `Queue` interface with:
- `map[string]*messageGroup` keyed by MessageGroupId
- Each group has its own `RingBuffer[*Message]` and `locked bool`
- Round-robin receive across unlocked groups
- Dedup map + expiry heap
- Visibility timeout per-message (shared inflight map)

- [ ] **Step 3: Run tests**

Run: `cd /Users/megan/cloudmock && go test ./services/sqs/ -run TestFIFO -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add services/sqs/fifo.go services/sqs/fifo_test.go
git commit -m "feat(sqs): FIFO queue with message groups and deduplication"
```

---

### Task 5: Dead-Letter Queue

**Files:**
- Create: `services/sqs/dlq.go`
- Create: `services/sqs/dlq_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/sqs/dlq_test.go`:

```go
package sqs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDLQ_MessageMovedAfterMaxReceives(t *testing.T) {
	main := NewStandardQueue("main", "http://localhost/queue/main", "arn:aws:sqs:us-east-1:000:main")
	dlq := NewStandardQueue("dlq", "http://localhost/queue/dlq", "arn:aws:sqs:us-east-1:000:dlq")
	defer main.Close()
	defer dlq.Close()

	main.SetDLQ(dlq, 2) // move to DLQ after 2 receives

	main.SendMessage("poison", 0, nil, "", "")

	// Receive 1 — visibility 1s
	msgs := main.ReceiveMessages(1, 1, 0)
	require.Len(t, msgs, 1)
	assert.Equal(t, 1, msgs[0].ReceiveCount)
	time.Sleep(1200 * time.Millisecond)

	// Receive 2 — exceeds maxReceiveCount, should be moved to DLQ
	msgs = main.ReceiveMessages(1, 1, 0)
	require.Len(t, msgs, 1)
	assert.Equal(t, 2, msgs[0].ReceiveCount)
	time.Sleep(1200 * time.Millisecond)

	// Message should now be in DLQ, not in main
	mainMsgs := main.ReceiveMessages(1, 30, 0)
	assert.Len(t, mainMsgs, 0)

	dlqMsgs := dlq.ReceiveMessages(1, 30, 0)
	require.Len(t, dlqMsgs, 1)
	assert.Equal(t, "poison", dlqMsgs[0].Body)
}
```

- [ ] **Step 2: Implement DLQ logic in dlq.go**

The DLQ logic hooks into the visibility timeout expiry path:
- When a message's visibility expires, increment `ReceiveCount`
- If `ReceiveCount >= maxReceiveCount` and DLQ is configured, send message to DLQ instead of back to ready queue

- [ ] **Step 3: Run tests**

Run: `cd /Users/megan/cloudmock && go test ./services/sqs/ -run TestDLQ -v -timeout 30s`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add services/sqs/dlq.go services/sqs/dlq_test.go
git commit -m "feat(sqs): dead-letter queue with redrive policy"
```

---

### Task 6: Store Rewrite + Handler Wiring

**Files:**
- Rewrite: `services/sqs/store.go`
- Modify: `services/sqs/handlers.go`
- Modify: `services/sqs/service.go`

- [ ] **Step 1: Rewrite store.go**

Add ARN index, support both StandardQueue and FIFOQueue via the `Queue` interface. Route CreateQueue to the right constructor based on `.fifo` suffix.

```go
type QueueStore struct {
	mu        sync.RWMutex
	byURL     map[string]Queue
	byName    map[string]Queue
	byARN     map[string]Queue
	accountID string
	region    string
}
```

- [ ] **Step 2: Update handlers to use Queue interface**

The handlers currently call `q.SendMessage(...)`, `q.ReceiveMessages(...)`, etc. These method signatures on the `Queue` interface match. Add `waitTimeSeconds` parameter support in `handleReceiveMessage`.

Add `RedrivePolicy` parsing in `handleSetQueueAttributes` — resolve DLQ ARN via store's `byARN` map.

- [ ] **Step 3: Run ALL existing tests**

Run: `cd /Users/megan/cloudmock && go test ./services/sqs/ -v`
Expected: ALL existing tests PASS (service_test.go, json_test.go)

- [ ] **Step 4: Commit**

```bash
git add services/sqs/store.go services/sqs/handlers.go services/sqs/service.go
git commit -m "feat(sqs): wire new queue types to handlers with long poll and DLQ support"
```

---

### Task 7: SQS Stress Benchmark

**Files:**
- Create: `benchmarks/suites/stress/sqs_stress.go`
- Create: `benchmarks/suites/stress/sqs_stress_test.go`

- [ ] **Step 1: Write stress benchmark**

Create `benchmarks/suites/stress/sqs_stress.go` with operations that:
- Send 100K messages, then receive all 100K
- 10K concurrent goroutines doing send/receive
- Long poll latency measurement (send triggers receive)
- FIFO throughput with 1K groups

- [ ] **Step 2: Create test**

```go
func TestSQSStressSuite_Metadata(t *testing.T) {
	s := NewSQSStressSuite()
	assert.Equal(t, "sqs-stress", s.Name())
}
```

- [ ] **Step 3: Run tests**

Run: `cd /Users/megan/cloudmock && go test ./benchmarks/suites/stress/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add benchmarks/suites/stress/sqs_stress*
git commit -m "feat(bench): add SQS scaling stress benchmark"
```

---

### Task 8: Benchmark and Validate Performance Targets

- [ ] **Step 1: Run SQS benchmark comparison**

```bash
go run ./benchmarks/cmd/bench --target=cloudmock --endpoint=http://localhost:4566 --services=sqs --iterations=100 --concurrency=10
```

- [ ] **Step 2: Run against LocalStack**

```bash
go run ./benchmarks/cmd/bench --target=localstack --endpoint=http://localhost:4567 --services=sqs --iterations=100 --concurrency=10
```

- [ ] **Step 3: Validate P50 targets**

Expected: SendMessage P50 < 0.05ms, ReceiveMessage P50 < 0.05ms

- [ ] **Step 4: Run Go benchmarks**

```bash
go test ./services/sqs/ -bench=. -benchmem
```

Expected: Send+Receive < 200ns/op

- [ ] **Step 5: Commit results**

```bash
git add benchmarks/results/
git commit -m "bench: SQS native performance results"
```
