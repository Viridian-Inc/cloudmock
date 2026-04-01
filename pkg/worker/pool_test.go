package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

func TestScheduleOnce_InstantMode(t *testing.T) {
	pool := NewPool(context.Background(), nil) // nil = default = instant
	defer pool.Shutdown()

	var called int32
	pool.ScheduleOnce("test", 5*time.Second, func() {
		atomic.AddInt32(&called, 1)
	})

	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected fn to be called once in instant mode, got %d", called)
	}
}

func TestScheduleInterval_InstantMode(t *testing.T) {
	pool := NewPool(context.Background(), nil)
	defer pool.Shutdown()

	var called int32
	pool.ScheduleInterval("test", 1*time.Second, func() {
		atomic.AddInt32(&called, 1)
	})

	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected fn to be called once in instant mode, got %d", called)
	}
}

func TestScheduleInterval_DelayedMode(t *testing.T) {
	cfg := lifecycle.DefaultConfig()
	cfg.SetEnabled(true)
	cfg.SetSpeedFactor(0.01) // 100x faster

	pool := NewPool(context.Background(), cfg)
	defer pool.Shutdown()

	var called int32
	// 1s interval at 0.01 speed = 10ms effective
	pool.ScheduleInterval("test", 1*time.Second, func() {
		atomic.AddInt32(&called, 1)
	})

	time.Sleep(60 * time.Millisecond)
	count := atomic.LoadInt32(&called)
	if count < 2 {
		t.Fatalf("expected fn to be called 2+ times in delayed mode, got %d", count)
	}
}

func TestScheduleOnce_Cancel(t *testing.T) {
	cfg := lifecycle.DefaultConfig()
	cfg.SetEnabled(true)
	cfg.SetSpeedFactor(1.0)

	pool := NewPool(context.Background(), cfg)
	defer pool.Shutdown()

	var called int32
	cancel := pool.ScheduleOnce("test", 1*time.Second, func() {
		atomic.AddInt32(&called, 1)
	})

	cancel()
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&called) != 0 {
		t.Fatal("fn should not have been called after cancel")
	}
}

func TestScheduleInterval_ReplaceByName(t *testing.T) {
	pool := NewPool(context.Background(), nil)
	defer pool.Shutdown()

	var first, second int32
	pool.ScheduleInterval("worker", 1*time.Second, func() {
		atomic.AddInt32(&first, 1)
	})
	time.Sleep(10 * time.Millisecond)

	// Replace with same name — first should stop
	pool.ScheduleInterval("worker", 1*time.Second, func() {
		atomic.AddInt32(&second, 1)
	})
	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&first) != 1 {
		t.Errorf("first worker should have been called once, got %d", first)
	}
	if atomic.LoadInt32(&second) != 1 {
		t.Errorf("second worker should have been called once, got %d", second)
	}
}

func TestShutdown_StopsAll(t *testing.T) {
	cfg := lifecycle.DefaultConfig()
	cfg.SetEnabled(true)
	cfg.SetSpeedFactor(0.01)

	pool := NewPool(context.Background(), cfg)

	var called int32
	pool.ScheduleInterval("a", 100*time.Millisecond, func() {
		atomic.AddInt32(&called, 1)
	})
	time.Sleep(10 * time.Millisecond)
	pool.Shutdown()

	before := atomic.LoadInt32(&called)
	time.Sleep(30 * time.Millisecond)
	after := atomic.LoadInt32(&called)

	if after > before+1 {
		t.Fatalf("worker should have stopped after shutdown, before=%d after=%d", before, after)
	}
}
