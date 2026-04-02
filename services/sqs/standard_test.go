package sqs

import (
	"testing"
	"time"
)

func TestStandardQueue_SendReceiveDelete(t *testing.T) {
	q := NewStandardQueue("test-q", "http://localhost/test-q", nil)
	defer q.Close()

	msgID := q.SendMessage("hello", 0, nil, "", "")
	if msgID == "" {
		t.Fatal("SendMessage returned empty ID")
	}

	msgs := q.ReceiveMessages(1, 30, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Body != "hello" {
		t.Errorf("expected body 'hello', got %q", msgs[0].Body)
	}
	if msgs[0].MessageId != msgID {
		t.Errorf("expected MessageId %s, got %s", msgID, msgs[0].MessageId)
	}

	ok := q.DeleteMessage(msgs[0].ReceiptHandle)
	if !ok {
		t.Error("DeleteMessage returned false")
	}

	// Should be empty now.
	msgs2 := q.ReceiveMessages(10, 30, 0)
	if len(msgs2) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs2))
	}
}

func TestStandardQueue_VisibilityTimeout(t *testing.T) {
	q := NewStandardQueue("vis-q", "http://localhost/vis-q", nil)
	defer q.Close()

	q.SendMessage("reappear", 0, nil, "", "")

	// Receive with 1s visibility timeout.
	msgs := q.ReceiveMessages(1, 1, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	// Immediately, should not be visible.
	msgs2 := q.ReceiveMessages(1, 30, 0)
	if len(msgs2) != 0 {
		t.Errorf("expected 0 messages while inflight, got %d", len(msgs2))
	}

	// Wait for visibility to expire.
	time.Sleep(1200 * time.Millisecond)

	// Should reappear.
	msgs3 := q.ReceiveMessages(1, 30, 0)
	if len(msgs3) != 1 {
		t.Fatalf("expected message to reappear after visibility timeout, got %d", len(msgs3))
	}
	if msgs3[0].Body != "reappear" {
		t.Errorf("expected body 'reappear', got %q", msgs3[0].Body)
	}
	if msgs3[0].ReceiveCount != 2 {
		t.Errorf("expected ReceiveCount 2, got %d", msgs3[0].ReceiveCount)
	}
}

func TestStandardQueue_DelayedMessage(t *testing.T) {
	q := NewStandardQueue("delay-q", "http://localhost/delay-q", nil)
	defer q.Close()

	q.SendMessage("delayed", 1, nil, "", "")

	// Should not be available immediately.
	msgs := q.ReceiveMessages(1, 30, 0)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages before delay, got %d", len(msgs))
	}

	// Wait for delay.
	time.Sleep(1200 * time.Millisecond)

	msgs2 := q.ReceiveMessages(1, 30, 0)
	if len(msgs2) != 1 {
		t.Fatalf("expected 1 message after delay, got %d", len(msgs2))
	}
	if msgs2[0].Body != "delayed" {
		t.Errorf("expected body 'delayed', got %q", msgs2[0].Body)
	}
}

func TestStandardQueue_Purge(t *testing.T) {
	q := NewStandardQueue("purge-q", "http://localhost/purge-q", nil)
	defer q.Close()

	for i := 0; i < 10; i++ {
		q.SendMessage("msg", 0, nil, "", "")
	}

	if n := q.ApproximateNumberOfMessages(); n != 10 {
		t.Errorf("expected 10 messages, got %d", n)
	}

	q.Purge()

	if n := q.ApproximateNumberOfMessages(); n != 0 {
		t.Errorf("expected 0 messages after purge, got %d", n)
	}
}

func TestStandardQueue_BatchSendReceive(t *testing.T) {
	q := NewStandardQueue("batch-q", "http://localhost/batch-q", nil)
	defer q.Close()

	for i := 0; i < 5; i++ {
		q.SendMessage("batch-msg", 0, nil, "", "")
	}

	msgs := q.ReceiveMessages(3, 30, 0)
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}

	msgs2 := q.ReceiveMessages(10, 30, 0)
	if len(msgs2) != 2 {
		t.Errorf("expected 2 remaining messages, got %d", len(msgs2))
	}
}

func BenchmarkStandardQueue_SendReceive(b *testing.B) {
	q := NewStandardQueue("bench-q", "http://localhost/bench-q", nil)
	defer q.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.SendMessage("benchmark message body", 0, nil, "", "")
		msgs := q.ReceiveMessages(1, 30, 0)
		if len(msgs) == 1 {
			q.DeleteMessage(msgs[0].ReceiptHandle)
		}
	}
}
