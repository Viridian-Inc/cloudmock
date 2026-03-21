package sqs

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MessageAttribute holds a single SQS message attribute.
type MessageAttribute struct {
	DataType    string
	StringValue string
	BinaryValue []byte
}

// Message represents an SQS message.
type Message struct {
	MessageId                   string
	Body                        string
	ReceiptHandle               string
	MD5OfBody                   string
	SentTimestamp               time.Time
	FirstReceiveTimestamp       time.Time
	ReceiveCount                int
	VisibilityDeadline          time.Time
	DelayUntil                  time.Time
	MessageAttributes           map[string]MessageAttribute
	MessageGroupId              string
	MessageDeduplicationId      string
}

// Queue is a single SQS queue with its messages and inflight tracking.
type Queue struct {
	mu                 sync.Mutex
	Name               string
	URL                string
	Attributes         map[string]string
	messages           []*Message
	inflight           map[string]*Message // keyed by ReceiptHandle
	IsFIFO             bool
	deduplicationCache map[string]time.Time // dedup ID → expiry
}

// newQueue creates a new Queue with sensible defaults.
func newQueue(name, url string, attrs map[string]string) *Queue {
	if attrs == nil {
		attrs = make(map[string]string)
	}
	q := &Queue{
		Name:               name,
		URL:                url,
		Attributes:         attrs,
		messages:           make([]*Message, 0),
		inflight:           make(map[string]*Message),
		IsFIFO:             strings.HasSuffix(name, ".fifo"),
		deduplicationCache: make(map[string]time.Time),
	}
	// Set default attributes if not provided.
	if _, ok := q.Attributes["VisibilityTimeout"]; !ok {
		q.Attributes["VisibilityTimeout"] = "30"
	}
	if _, ok := q.Attributes["MessageRetentionPeriod"]; !ok {
		q.Attributes["MessageRetentionPeriod"] = "345600" // 4 days
	}
	if _, ok := q.Attributes["MaximumMessageSize"]; !ok {
		q.Attributes["MaximumMessageSize"] = "262144" // 256 KB
	}
	if _, ok := q.Attributes["DelaySeconds"]; !ok {
		q.Attributes["DelaySeconds"] = "0"
	}
	if _, ok := q.Attributes["ReceiveMessageWaitTimeSeconds"]; !ok {
		q.Attributes["ReceiveMessageWaitTimeSeconds"] = "0"
	}
	if q.IsFIFO {
		q.Attributes["FifoQueue"] = "true"
		if _, ok := q.Attributes["ContentBasedDeduplication"]; !ok {
			q.Attributes["ContentBasedDeduplication"] = "false"
		}
	}
	return q
}

// SendMessage enqueues a message and returns its MessageId. Returns empty string
// if a FIFO duplicate is detected.
func (q *Queue) SendMessage(body string, delaySeconds int, attrs map[string]MessageAttribute, groupID, dedupID string) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()

	// FIFO deduplication.
	if q.IsFIFO && dedupID != "" {
		q.pruneDeduplicationCache(now)
		if _, dup := q.deduplicationCache[dedupID]; dup {
			// Return the original message's ID — for simplicity return a stable
			// placeholder so the caller knows it was a duplicate; the real AWS
			// returns the original MessageId, but we don't track that here.
			return ""
		}
		q.deduplicationCache[dedupID] = now.Add(5 * time.Minute)
	}

	msgID := newUUID()
	delayUntil := now.Add(time.Duration(delaySeconds) * time.Second)

	msg := &Message{
		MessageId:              msgID,
		Body:                   body,
		MD5OfBody:              md5Hex(body),
		SentTimestamp:          now,
		DelayUntil:             delayUntil,
		MessageAttributes:      attrs,
		MessageGroupId:         groupID,
		MessageDeduplicationId: dedupID,
	}
	q.messages = append(q.messages, msg)
	return msgID
}

// ReceiveMessages returns up to maxCount messages that are available (past their
// delay, not in-flight). Moved to inflight with a new ReceiptHandle and visibility
// deadline.
func (q *Queue) ReceiveMessages(maxCount int, visibilityTimeout int) []*Message {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()

	// Reclaim expired inflight messages back to the available pool.
	for rh, msg := range q.inflight {
		if now.After(msg.VisibilityDeadline) {
			delete(q.inflight, rh)
			q.messages = append(q.messages, msg)
		}
	}

	result := make([]*Message, 0, maxCount)
	remaining := make([]*Message, 0, len(q.messages))

	for _, msg := range q.messages {
		if len(result) >= maxCount {
			remaining = append(remaining, msg)
			continue
		}
		if now.Before(msg.DelayUntil) {
			remaining = append(remaining, msg)
			continue
		}
		// Assign new ReceiptHandle and move to inflight.
		rh := newReceiptHandle()
		msg.ReceiptHandle = rh
		msg.ReceiveCount++
		if msg.ReceiveCount == 1 {
			msg.FirstReceiveTimestamp = now
		}
		msg.VisibilityDeadline = now.Add(time.Duration(visibilityTimeout) * time.Second)
		q.inflight[rh] = msg
		result = append(result, msg)
	}
	q.messages = remaining
	return result
}

// DeleteMessage removes a message from inflight by ReceiptHandle.
// Returns false if the ReceiptHandle is not found.
func (q *Queue) DeleteMessage(receiptHandle string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.inflight[receiptHandle]; ok {
		delete(q.inflight, receiptHandle)
		return true
	}
	return false
}

// ChangeMessageVisibility updates the visibility deadline for an inflight message.
// Returns false if not found.
func (q *Queue) ChangeMessageVisibility(receiptHandle string, visibilityTimeout int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	msg, ok := q.inflight[receiptHandle]
	if !ok {
		return false
	}
	msg.VisibilityDeadline = time.Now().UTC().Add(time.Duration(visibilityTimeout) * time.Second)
	return true
}

// Purge removes all available and inflight messages.
func (q *Queue) Purge() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = make([]*Message, 0)
	q.inflight = make(map[string]*Message)
}

// ApproximateNumberOfMessages returns the current available message count.
func (q *Queue) ApproximateNumberOfMessages() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages)
}

// ApproximateNumberOfMessagesNotVisible returns the inflight message count.
func (q *Queue) ApproximateNumberOfMessagesNotVisible() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.inflight)
}

// pruneDeduplicationCache removes expired entries. Caller must hold q.mu.
func (q *Queue) pruneDeduplicationCache(now time.Time) {
	for id, exp := range q.deduplicationCache {
		if now.After(exp) {
			delete(q.deduplicationCache, id)
		}
	}
}

// ---- helpers ----

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newReceiptHandle() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
