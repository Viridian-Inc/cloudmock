package lifecycle

import (
	"testing"
	"time"
)

func TestInstantTransitions(t *testing.T) {
	transitions := []Transition{
		{From: "CREATING", To: "AVAILABLE", Delay: 30 * time.Second},
		{From: "DELETING", To: "DELETED", Delay: 10 * time.Second},
	}

	// Default config: delays disabled → instant transitions.
	m := NewMachine("CREATING", transitions, nil)

	if m.State() != "AVAILABLE" {
		t.Fatalf("expected AVAILABLE, got %s", m.State())
	}

	m.ForceState("DELETING")
	if m.State() != "DELETED" {
		t.Fatalf("expected DELETED, got %s", m.State())
	}

	history := m.History()
	if len(history) != 4 {
		t.Fatalf("expected 4 history records, got %d", len(history))
	}
	expected := []State{"CREATING", "AVAILABLE", "DELETING", "DELETED"}
	for i, rec := range history {
		if rec.State != expected[i] {
			t.Errorf("history[%d] = %s, want %s", i, rec.State, expected[i])
		}
	}
}

func TestDelayedTransitions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetEnabled(true)
	cfg.SetSpeedFactor(0.01) // 100x faster for testing

	transitions := []Transition{
		{From: "CREATING", To: "AVAILABLE", Delay: 1 * time.Second}, // effective: 10ms
	}

	m := NewMachine("CREATING", transitions, cfg)
	defer m.Stop()

	if m.State() != "CREATING" {
		t.Fatalf("expected CREATING immediately, got %s", m.State())
	}

	// Wait for the transition.
	time.Sleep(50 * time.Millisecond)
	if m.State() != "AVAILABLE" {
		t.Fatalf("expected AVAILABLE after delay, got %s", m.State())
	}
}

func TestTerminalState(t *testing.T) {
	transitions := []Transition{
		{From: "CREATING", To: "ACTIVE", Delay: 0},
	}

	m := NewMachine("CREATING", transitions, nil)
	if m.State() != "ACTIVE" {
		t.Fatalf("expected ACTIVE, got %s", m.State())
	}

	// ACTIVE is terminal (no transition defined from it).
	history := m.History()
	if len(history) != 2 {
		t.Fatalf("expected 2 history records, got %d", len(history))
	}
}

func TestOnTransitionCallback(t *testing.T) {
	transitions := []Transition{
		{From: "PENDING", To: "RUNNING", Delay: 0},
	}

	var called bool
	var fromState, toState State

	m := NewMachine("PENDING", transitions, nil)
	// Set callback — note: instant transitions in NewMachine fire before we
	// can set the callback, so test with ForceState.
	m.OnTransition(func(from, to State) {
		called = true
		fromState = from
		toState = to
	})

	m.ForceState("STOPPING")
	time.Sleep(10 * time.Millisecond) // allow goroutine

	if !called {
		t.Fatal("callback was not invoked")
	}
	if fromState != "RUNNING" || toState != "STOPPING" {
		t.Errorf("callback got from=%s to=%s, want RUNNING→STOPPING", fromState, toState)
	}
}
