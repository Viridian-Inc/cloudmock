package sqs

import (
	"testing"
	"time"
)

func TestDLQ_MessageMovedAfterMaxReceives(t *testing.T) {
	src := NewStandardQueue("src-q", "http://localhost/src-q", nil)
	defer src.Close()

	dlq := NewStandardQueue("dlq-q", "http://localhost/dlq-q", nil)
	defer dlq.Close()

	// Configure DLQ with maxReceiveCount=2.
	src.SetDLQ(dlq, 2)

	src.SendMessage("dlq-candidate", 0, nil, "", "")

	// First receive (receiveCount=1).
	msgs := src.ReceiveMessages(1, 1, 0)
	if len(msgs) != 1 {
		t.Fatalf("first receive: expected 1, got %d", len(msgs))
	}

	// Wait for visibility to expire.
	time.Sleep(1200 * time.Millisecond)

	// Second receive (receiveCount=2 — matches maxReceiveCount).
	msgs2 := src.ReceiveMessages(1, 1, 0)
	if len(msgs2) != 1 {
		t.Fatalf("second receive: expected 1, got %d", len(msgs2))
	}

	// Wait for visibility to expire again.
	time.Sleep(1200 * time.Millisecond)

	// Third receive attempt — message should have been moved to DLQ
	// (receiveCount >= maxReceiveCount on reclaim).
	msgs3 := src.ReceiveMessages(1, 30, 0)
	if len(msgs3) != 0 {
		t.Errorf("third receive from source: expected 0 (moved to DLQ), got %d", len(msgs3))
	}

	// Check DLQ has the message.
	dlqMsgs := dlq.ReceiveMessages(10, 30, 0)
	if len(dlqMsgs) != 1 {
		t.Fatalf("DLQ: expected 1 message, got %d", len(dlqMsgs))
	}
	if dlqMsgs[0].Body != "dlq-candidate" {
		t.Errorf("DLQ: expected body 'dlq-candidate', got %q", dlqMsgs[0].Body)
	}
}
