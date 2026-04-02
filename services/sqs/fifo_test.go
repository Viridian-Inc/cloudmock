package sqs

import (
	"testing"
)

func TestFIFO_OrderPreserved(t *testing.T) {
	q := NewFIFOQueue("order.fifo", "http://localhost/order.fifo", nil)
	defer q.Close()

	bodies := []string{"first", "second", "third"}
	for i, body := range bodies {
		id := q.SendMessage(body, 0, nil, "grp1", newUUID())
		if id == "" {
			t.Fatalf("SendMessage %d returned empty ID", i)
		}
	}

	msgs := q.ReceiveMessages(10, 30, 0)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	for i, want := range bodies {
		if msgs[i].Body != want {
			t.Errorf("message %d: expected body %q, got %q", i, want, msgs[i].Body)
		}
	}
}

func TestFIFO_GroupIsolation(t *testing.T) {
	q := NewFIFOQueue("isolation.fifo", "http://localhost/isolation.fifo", nil)
	defer q.Close()

	// Send to two groups.
	q.SendMessage("A1", 0, nil, "groupA", newUUID())
	q.SendMessage("B1", 0, nil, "groupB", newUUID())
	q.SendMessage("A2", 0, nil, "groupA", newUUID())

	// Receive one from each group (both groups unlocked initially).
	msgs := q.ReceiveMessages(10, 30, 0)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages from initial receive, got %d", len(msgs))
	}

	// Delete all messages to unlock groups.
	for _, m := range msgs {
		q.DeleteMessage(m.ReceiptHandle)
	}

	// Now send again and receive just 1 — should lock that group.
	q.SendMessage("A3", 0, nil, "groupA", newUUID())
	q.SendMessage("B2", 0, nil, "groupB", newUUID())
	q.SendMessage("A4", 0, nil, "groupA", newUUID())

	msgs2 := q.ReceiveMessages(1, 30, 0)
	if len(msgs2) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs2))
	}
	lockedGroup := msgs2[0].MessageGroupId

	// Try to receive more — should get from the OTHER group only (locked group blocks).
	msgs3 := q.ReceiveMessages(10, 30, 0)
	for _, m := range msgs3 {
		if m.MessageGroupId == lockedGroup {
			t.Errorf("received message from locked group %q", lockedGroup)
		}
	}
}

func TestFIFO_Deduplication(t *testing.T) {
	q := NewFIFOQueue("dedup.fifo", "http://localhost/dedup.fifo", nil)
	defer q.Close()

	id1 := q.SendMessage("original", 0, nil, "grp", "dedup-key")
	if id1 == "" {
		t.Fatal("first SendMessage returned empty ID")
	}

	id2 := q.SendMessage("duplicate", 0, nil, "grp", "dedup-key")
	if id2 != "" {
		t.Errorf("expected empty ID for duplicate, got %q", id2)
	}

	msgs := q.ReceiveMessages(10, 30, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after dedup, got %d", len(msgs))
	}
	if msgs[0].Body != "original" {
		t.Errorf("expected body 'original', got %q", msgs[0].Body)
	}
}
