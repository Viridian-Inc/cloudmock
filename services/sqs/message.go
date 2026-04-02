package sqs

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
