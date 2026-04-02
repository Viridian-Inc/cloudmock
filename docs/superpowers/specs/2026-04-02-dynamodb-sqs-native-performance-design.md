# Native-Performance DynamoDB & SQS — Design Spec

## Goal

Rewrite the DynamoDB and SQS service internals in CloudMock to achieve sub-millisecond P50 latency for single-item ops and flat scaling to 10M+ items / 100K+ messages under 10K concurrent goroutines. Full DynamoDB query semantics and full SQS feature set (FIFO, long polling, DLQ).

## Performance Targets

| Operation | Target P50 | Current P50 | Scaling |
|-----------|-----------|-------------|---------|
| DynamoDB GetItem | <0.1ms | 2-5ms | O(1) at 10M items |
| DynamoDB PutItem | <0.2ms | 2-5ms | O(1) + O(log n) index |
| DynamoDB DeleteItem | <0.1ms | 2-5ms | O(1) + O(log n) index |
| DynamoDB UpdateItem | <0.2ms | 2-5ms | O(1) + O(log n) index |
| DynamoDB Query (100 results) | <1ms | 3-5ms | O(log n + k) where k=results |
| DynamoDB Scan (1MB page) | <5ms | 3-5ms | O(page_size) |
| DynamoDB BatchWriteItem (25) | <0.5ms | N/A | O(25) |
| SQS SendMessage | <0.05ms | 2-3ms | O(1) amortized |
| SQS ReceiveMessage | <0.05ms | 2-3ms | O(1) |
| SQS Long Poll (msg arrives) | <1ms | N/A | O(1) wake |
| SQS DeleteMessage | <0.05ms | 2-3ms | O(1) |
| SQS FIFO Send | <0.1ms | N/A | O(1) per group |
| SQS FIFO Receive | <0.1ms | N/A | O(1) per group |

All targets must hold at 10K concurrent goroutines.

---

## DynamoDB Storage Engine

### Current Problems

1. **Items in flat `[]Item` slice** — GetItem/DeleteItem/UpdateItem are O(n) linear scans
2. **No primary key index** — every single-item op scans the full table
3. **GSI/LSI removal is linear scan** — `removeFromIndex()` walks entire index
4. **Expression parsing per-item** — `resolveNames()` does `strings.ReplaceAll` per evaluation
5. **TTL reaper** — full table scan every 5s holding write lock
6. **Single `sync.RWMutex` per store** — all tables contend on one lock

### New Architecture

#### Primary Storage: Hash + B-Tree

```
Table
├── meta            TableMeta (name, key schema, GSI/LSI defs, TTL config)
├── mu              sync.RWMutex (per-table, not per-store)
├── partitions      map[string]*Partition  // partitionKey -> partition
│   └── Partition
│       ├── items   *btree.BTreeG[Item]    // sorted by sort key (tidwall/btree)
│       └── count   int64                  // atomic, for DescribeTable
├── pkIndex         map[string]*Item       // serialized(pk+sk) -> item pointer (O(1) Get)
├── gsi             map[string]*IndexStore // GSI name -> index
├── lsi             map[string]*IndexStore // LSI name -> index
└── ttlHeap         *ttlMinHeap           // min-heap ordered by TTL timestamp
```

**For tables without sort key:** `Partition.items` is nil, single item stored directly. GetItem is pure hash lookup via `pkIndex`.

**For tables with sort key:** `pkIndex` gives O(1) exact match. Range queries (Query with `begins_with`, `BETWEEN`, `<`, `>`) use `Partition.items` B-tree for O(log n + k) scans where k = result count.

#### Index Storage (GSI/LSI)

```
IndexStore
├── partitions   map[string]*btree.BTreeG[Item]  // index PK -> sorted by index SK
└── pkIndex      map[string]*Item                 // serialized(indexPK+indexSK) -> item
```

Maintained on write:
- PutItem: insert into all applicable indexes
- DeleteItem: remove from all applicable indexes (O(log n) per index via B-tree delete)
- UpdateItem: remove old entry, insert new (only if indexed attributes changed)

#### Expression Engine

Replace string-based expression evaluation with a compiled AST:

```go
type ExprNode interface {
    Evaluate(item Item) bool  // for conditions
    Project(item Item) Item   // for projections
}

type exprCache struct {
    mu    sync.RWMutex
    exprs map[string]ExprNode  // expression string -> compiled AST
}
```

**Compilation:** Parse `KeyConditionExpression`, `FilterExpression`, `ProjectionExpression`, `ConditionExpression`, and `UpdateExpression` into typed AST nodes. Cache by expression string + attribute name mapping.

**Evaluation:** Direct struct field access, typed comparisons (string vs string, number vs number using `math/big` only when needed for precision). No string allocations in the hot path.

**Supported operators:**
- Comparison: `=`, `<>`, `<`, `<=`, `>`, `>=`
- Functions: `attribute_exists`, `attribute_not_exists`, `attribute_type`, `begins_with`, `contains`, `size`
- Logical: `AND`, `OR`, `NOT`
- Between: `BETWEEN`
- In: `IN`
- Update: `SET`, `REMOVE`, `ADD`, `DELETE`

#### TTL Reaper

```go
type ttlMinHeap struct {
    mu    sync.Mutex
    items []ttlEntry  // min-heap by expiry time
    timer *time.Timer // fires at next expiry
}

type ttlEntry struct {
    key    string    // serialized pk+sk
    expiry time.Time
}
```

- On PutItem with TTL attribute: push entry to heap, reset timer if this is the new minimum
- Timer fires: pop all expired entries, delete items (acquiring table write lock briefly per batch of 25)
- No polling, no full scans. O(log n) per insert/delete.

#### Concurrency Model

- **Store level:** `sync.RWMutex` for table creation/deletion/listing only
- **Table level:** `sync.RWMutex` per table. Reads (Get/Query/Scan) take RLock. Writes (Put/Update/Delete) take Lock.
- **Atomic counters:** `atomic.Int64` for item counts, table size estimates
- **No global lock contention:** Operations on different tables are fully concurrent

#### Pagination

- Query/Scan return up to 1MB of data or `Limit` items (whichever comes first)
- `LastEvaluatedKey` is the key of the last returned item
- `ExclusiveStartKey` resumes the B-tree iterator from that position — O(log n) seek

#### Transactions

- `TransactWriteItems`: Acquire write locks on all involved tables (sorted by table name to prevent deadlock), validate all conditions, apply all writes atomically, release locks
- `TransactGetItems`: Acquire read locks on all involved tables, read items, release locks
- Maximum 100 items per transaction (matching AWS limit)

---

## SQS Queue Engine

### Current Problems

1. **ReceiveMessage iterates ALL messages** — O(n) with full slice rebuild
2. **Dedup cache pruned by full scan** on every send
3. **No delayed message handling** — scans all messages checking `DelayUntil`
4. **No long polling** — no blocking receive
5. **No DLQ support**

### New Architecture

#### Standard Queue

```
StandardQueue
├── meta          QueueMeta (name, URL, attributes, DLQ config)
├── ready         *RingBuffer[*Message]     // O(1) send/receive
├── delayed       *MinHeap[*Message]        // by DelayUntil, timer-driven
├── inflight      map[string]*inflightMsg   // receiptHandle -> msg + visibility
├── visTimeout    *MinHeap[*inflightMsg]    // by visibility expiry, timer-driven
├── delayTimer    *time.Timer               // fires at next delay expiry
├── visTimer      *time.Timer               // fires at next visibility expiry
├── waiters       []chan []*Message          // long-poll waiters
├── mu            sync.Mutex                // structural lock
├── dlq           *StandardQueue            // dead-letter target (nil if none)
├── maxReceives   int                       // redrive policy threshold
└── msgCount      atomic.Int64              // approximate message count
```

#### Ring Buffer

```go
type RingBuffer[T any] struct {
    buf  []T
    head int
    tail int
    len  int
    cap  int
}
```

- `Push(item T)`: write at tail, advance tail, grow if full (double capacity). O(1) amortized.
- `Pop() (T, bool)`: read from head, advance head. O(1).
- `Len() int`: O(1).
- Pre-allocate 1024 slots, grow to 1M+.

#### Delayed Messages

- SendMessage with `DelaySeconds > 0`: push to `delayed` min-heap
- `delayTimer` fires at the earliest delay: pop all matured messages, push to `ready` ring buffer
- Reset timer to next delay. O(log n) per insert, O(1) amortized per maturation.

#### Visibility Timeout

- ReceiveMessage: pop from `ready`, create `inflightMsg` with `visibleAt = now + visibilityTimeout`, push to `visTimeout` heap, return message
- `visTimer` fires: pop all expired inflight messages, push back to `ready` (incrementing `receiveCount`), check DLQ threshold
- DeleteMessage: remove from `inflight` map (O(1)), mark as deleted in vis heap (lazy deletion)
- ChangeMessageVisibility: update `visibleAt` in inflight, reposition in vis heap

#### FIFO Queue

```
FIFOQueue
├── meta           QueueMeta
├── groups         map[string]*MessageGroup  // MessageGroupId -> group
├── groupOrder     []string                  // round-robin order for receive
├── groupIdx       int                       // current position in round-robin
├── dedup          map[string]time.Time      // deduplicationId -> expiry
├── dedupHeap      *MinHeap[dedupEntry]      // by expiry time, timer-driven prune
├── inflight       map[string]*inflightMsg
├── visTimeout     *MinHeap[*inflightMsg]
├── visTimer       *time.Timer
├── waiters        []chan []*Message
├── mu             sync.Mutex
└── dlq            *StandardQueue
```

**MessageGroup:**
```go
type MessageGroup struct {
    id       string
    messages *RingBuffer[*Message]
    locked   bool  // true while a message from this group is inflight
}
```

- FIFO ordering: within a group, messages are strictly ordered. `locked = true` prevents next message from being received until current is deleted or times out.
- Receive round-robins across unlocked groups.
- Dedup: on send, check `dedup` map. If exists and within 5-minute window, return previous MessageId. Expiry heap prunes old entries — no full scan.

#### Long Polling

```go
func (q *StandardQueue) ReceiveMessage(maxMsgs int, waitTimeSec int) []*Message {
    q.mu.Lock()
    msgs := q.tryReceive(maxMsgs)
    if len(msgs) > 0 || waitTimeSec == 0 {
        q.mu.Unlock()
        return msgs
    }

    // Register waiter
    ch := make(chan []*Message, 1)
    q.waiters = append(q.waiters, ch)
    q.mu.Unlock()

    // Block until messages arrive or timeout
    select {
    case msgs := <-ch:
        return msgs
    case <-time.After(time.Duration(waitTimeSec) * time.Second):
        q.removeWaiter(ch)
        return nil
    }
}

func (q *StandardQueue) SendMessage(msg *Message) {
    q.mu.Lock()
    q.ready.Push(msg)
    // Wake a waiter if any
    if len(q.waiters) > 0 {
        ch := q.waiters[0]
        q.waiters = q.waiters[1:]
        msgs := q.tryReceive(10) // default batch
        q.mu.Unlock()
        ch <- msgs
        return
    }
    q.mu.Unlock()
}
```

#### Dead-Letter Queue

- Each queue can configure `RedrivePolicy`: `{"deadLetterTargetArn": "...", "maxReceiveCount": N}`
- When a message's `receiveCount` exceeds `maxReceiveCount`, it's moved to the DLQ instead of back to ready
- `RedriveAllowPolicy` controls which source queues can target this DLQ

#### Queue Store

```go
type QueueStore struct {
    mu     sync.RWMutex
    byURL  map[string]Queue       // interface: StandardQueue or FIFOQueue
    byName map[string]Queue
    byARN  map[string]Queue       // for DLQ resolution
}
```

---

## Testing Strategy

### Correctness Tests

**DynamoDB:**
- Every comparison operator with every attribute type (S, N, B, SS, NS, BS, L, M, BOOL, NULL)
- KeyConditionExpression: all operators on hash key, range key, begins_with, BETWEEN
- FilterExpression: attribute_exists, attribute_not_exists, contains, size, AND/OR/NOT
- UpdateExpression: SET (path, if_not_exists, list_append), REMOVE, ADD (number, set), DELETE (set)
- ProjectionExpression: top-level, nested paths
- Pagination: ExclusiveStartKey/LastEvaluatedKey across Query and Scan
- GSI/LSI: query on secondary indexes, project non-key attributes
- Transactions: TransactWriteItems with conditions, TransactGetItems
- TTL: items expire and are removed without affecting reads
- ConditionalCheckFailedException on failed conditions

**SQS:**
- Standard: send, receive, delete, visibility timeout, change visibility
- FIFO: message group ordering, exactly-once dedup within 5-min window
- Long polling: blocks until message arrives, times out correctly
- DLQ: message moves after maxReceiveCount exceeded
- Batch operations: SendMessageBatch, ReceiveMessage with MaxNumberOfMessages, DeleteMessageBatch
- Queue attributes: ApproximateNumberOfMessages, ApproximateNumberOfMessagesNotVisible

### Performance Tests

**Scaling benchmarks** (added to existing harness with `--stress` flag):

```
DynamoDB:
  - GetItem at 1K, 10K, 100K, 1M, 10M items — assert P50 < 0.1ms at all sizes
  - PutItem sustained throughput — 10K goroutines, measure ops/sec
  - Query with 100 results at 1M items — assert P50 < 1ms
  - Mixed read/write (80/20) at 10K goroutines — assert P99 < 5ms

SQS:
  - SendMessage at 100K queue depth — assert P50 < 0.05ms
  - ReceiveMessage at 100K queue depth — assert P50 < 0.05ms
  - Long poll latency — send triggers receive within 1ms
  - FIFO throughput — 1K groups, 10K goroutines, measure ops/sec
  - Visibility timeout accuracy — message reappears within 100ms of timeout
```

### Benchmark Harness Integration

Add to `benchmarks/cmd/bench`:
- `--stress` flag: runs large-dataset benchmarks (1M items, 100K messages)
- `--scale-test` flag: runs at 1K/10K/100K/1M/10M and reports scaling curve
- New stress suites: `benchmarks/suites/stress/dynamodb_stress.go`, `sqs_stress.go`

---

## Files Changed

### DynamoDB (new/modified files)

| File | Action | Purpose |
|------|--------|---------|
| `services/dynamodb/store.go` | **Rewrite** | Table store with per-table locks |
| `services/dynamodb/table.go` | **Rewrite** | Partition map + B-tree + pkIndex |
| `services/dynamodb/partition.go` | **New** | Partition struct with B-tree |
| `services/dynamodb/expression.go` | **Rewrite** | AST-based expression compiler + cache |
| `services/dynamodb/expression_ast.go` | **New** | AST node types and evaluator |
| `services/dynamodb/ttl.go` | **Rewrite** | Heap-based TTL reaper |
| `services/dynamodb/transaction.go` | **Modify** | Use indexed lookups, sorted lock acquisition |
| `services/dynamodb/service.go` | **Modify** | Wire new store, minimal changes |
| `services/dynamodb/*_test.go` | **New** | Comprehensive correctness + perf tests |

### SQS (new/modified files)

| File | Action | Purpose |
|------|--------|---------|
| `services/sqs/store.go` | **Rewrite** | Queue store with ARN index |
| `services/sqs/queue.go` | **Delete** | Replaced by standard.go + fifo.go |
| `services/sqs/standard.go` | **New** | Ring buffer queue with timers |
| `services/sqs/fifo.go` | **New** | FIFO queue with message groups |
| `services/sqs/ringbuffer.go` | **New** | Generic ring buffer |
| `services/sqs/minheap.go` | **New** | Generic min-heap for delays/visibility |
| `services/sqs/longpoll.go` | **New** | Waiter management for long polling |
| `services/sqs/dlq.go` | **New** | Dead-letter queue redrive logic |
| `services/sqs/service.go` | **Modify** | Wire new queue types |
| `services/sqs/*_test.go` | **New** | Comprehensive correctness + perf tests |

### Shared

| File | Action | Purpose |
|------|--------|---------|
| `pkg/collections/btree.go` | **New** | Thin wrapper around tidwall/btree |
| `pkg/collections/ringbuffer.go` | **New** | Generic ring buffer (shared) |
| `pkg/collections/minheap.go` | **New** | Generic min-heap (shared) |
| `benchmarks/suites/stress/dynamodb_stress.go` | **New** | Scaling benchmarks |
| `benchmarks/suites/stress/sqs_stress.go` | **New** | Scaling benchmarks |
| `benchmarks/cmd/bench/main.go` | **Modify** | Add --stress and --scale-test flags |

### External dependency

- `github.com/tidwall/btree` — high-performance generic B-tree for Go. Zero-allocation iteration, 10-50ns per op.

---

## Build Order

1. Shared collections (ring buffer, min-heap) — no dependencies, fully testable in isolation
2. DynamoDB expression AST compiler — can test against current expression tests
3. DynamoDB storage engine (partitions, B-tree, pkIndex) — core rewrite
4. DynamoDB TTL reaper — heap-based replacement
5. DynamoDB transactions — adapt to new storage
6. DynamoDB correctness tests + performance tests
7. SQS ring buffer + min-heap wiring
8. SQS standard queue (ready, delayed, inflight, visibility)
9. SQS long polling
10. SQS FIFO queue + deduplication
11. SQS dead-letter queue
12. SQS correctness tests + performance tests
13. Benchmark harness stress mode integration
14. Final benchmark: CloudMock vs LocalStack at scale
