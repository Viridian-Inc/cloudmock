package sqs

import (
	"testing"
	"time"
)

func TestLongPoll_MessageArrivesWhileWaiting(t *testing.T) {
	q := NewStandardQueue("lp-arrive-q", "http://localhost/lp-arrive-q", nil)
	defer q.Close()

	resultCh := make(chan []*Message, 1)

	// Start a goroutine that blocks on receive with long polling.
	go func() {
		msgs := q.ReceiveMessages(1, 30, 5)
		resultCh <- msgs
	}()

	// Give the goroutine time to enter the wait state.
	time.Sleep(100 * time.Millisecond)

	// Send a message — should wake the waiter.
	q.SendMessage("wakeup", 0, nil, "", "")

	select {
	case msgs := <-resultCh:
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		if msgs[0].Body != "wakeup" {
			t.Errorf("expected body 'wakeup', got %q", msgs[0].Body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for long-poll to return")
	}
}

func TestLongPoll_TimeoutReturnsEmpty(t *testing.T) {
	q := NewStandardQueue("lp-timeout-q", "http://localhost/lp-timeout-q", nil)
	defer q.Close()

	start := time.Now()
	msgs := q.ReceiveMessages(1, 30, 1)
	elapsed := time.Since(start)

	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}

	// Should have waited approximately 1 second.
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected ~1s wait, got %v", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("waited too long: %v", elapsed)
	}
}
