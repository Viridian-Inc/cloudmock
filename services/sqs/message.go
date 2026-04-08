package sqs

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync/atomic"
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
	MessageId              string
	Body                   string
	ReceiptHandle          string
	MD5OfBody              string
	SentTimestamp          time.Time
	FirstReceiveTimestamp  time.Time
	ReceiveCount           int
	VisibilityDeadline     time.Time
	DelayUntil             time.Time
	MessageAttributes      map[string]MessageAttribute
	MessageGroupId         string
	MessageDeduplicationId string
}

// Queue is the interface for all SQS queue implementations (standard and FIFO).
type Queue interface {
	QueueName() string
	QueueURL() string
	IsFIFOQueue() bool
	GetAttributes() map[string]string
	SetAttributes(attrs map[string]string)
	SendMessage(body string, delaySeconds int, attrs map[string]MessageAttribute, groupID, dedupID string) string
	ReceiveMessages(maxCount int, visibilityTimeout int, waitTimeSeconds int) []*Message
	DeleteMessage(receiptHandle string) bool
	ChangeMessageVisibility(receiptHandle string, timeout int) bool
	Purge()
	ApproximateNumberOfMessages() int
	ApproximateNumberOfMessagesNotVisible() int
	SetDLQ(target Queue, maxReceiveCount int)
	Close()
}

// ---- helpers ----

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// uuidCounter is an atomic counter for fast UUID generation.
// Combined with a timestamp prefix, produces unique IDs without crypto/rand.
var uuidCounter atomic.Uint64

func newUUID() string {
	n := uuidCounter.Add(1)
	t := uint64(time.Now().UnixNano())
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		t>>32, (t>>16)&0xFFFF, n&0xFFFF, (n>>16)&0xFFFF, t&0xFFFFFFFFFFFF)
}

// receiptCounter generates receipt handles without crypto/rand.
var receiptCounter atomic.Uint64

func newReceiptHandle() string {
	n := receiptCounter.Add(1)
	t := uint64(time.Now().UnixNano())
	// 32 hex chars = 16 bytes equivalent, matching original format.
	return fmt.Sprintf("%016x%016x", t, n)
}
