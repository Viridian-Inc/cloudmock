package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
)

func TestScheduleOnce_InstantMode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := NewPool(context.Background(), nil) // nil = default = instant
		defer pool.Shutdown()
		var called atomic.Int32
		pool.ScheduleOnce("test", 5*time.Second, func() { called.Add(1) })
		synctest.Wait()
		if got := called.Load(); got != 1 {
			t.Fatalf("instant callbacks = %d, want 1", got)
		}
	})
}

func TestScheduleInterval_InstantMode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pool := NewPool(context.Background(), nil)
		defer pool.Shutdown()
		var called atomic.Int32
		pool.ScheduleInterval("test", time.Second, func() { called.Add(1) })
		synctest.Wait()
		time.Sleep(3 * time.Second)
		synctest.Wait()
		if got := called.Load(); got != 1 {
			t.Fatalf("instant interval callbacks = %d, want only the initial call", got)
		}
	})
}

func TestScheduleInterval_DelayedMode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := lifecycle.DefaultConfig()
		cfg.SetEnabled(true)
		cfg.SetSpeedFactor(0.01) // 1s interval becomes 10ms effective.
		pool := NewPool(context.Background(), cfg)
		defer pool.Shutdown()
		var called atomic.Int32
		pool.ScheduleInterval("test", time.Second, func() { called.Add(1) })
		synctest.Wait()
		if got := called.Load(); got != 1 {
			t.Fatalf("initial callbacks = %d, want 1", got)
		}
		time.Sleep(25 * time.Millisecond)
		synctest.Wait()
		if got := called.Load(); got != 3 {
			t.Fatalf("interval callbacks = %d, want initial call and two ticks", got)
		}
	})
}

func TestScheduleOnce_DelayedMode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := lifecycle.DefaultConfig()
		cfg.SetEnabled(true)
		pool := NewPool(context.Background(), cfg)
		defer pool.Shutdown()
		var called atomic.Int32
		pool.ScheduleOnce("test", time.Second, func() { called.Add(1) })
		synctest.Wait()
		if got := called.Load(); got != 0 {
			t.Fatalf("callbacks before delay = %d, want 0", got)
		}
		time.Sleep(3 * time.Second)
		synctest.Wait()
		if got := called.Load(); got != 1 {
			t.Fatalf("callbacks after delay = %d, want 1", got)
		}
	})
}

func TestScheduleOnce_Cancel(t *testing.T) {
	for _, method := range []string{"returned", "named", "parent"} {
		t.Run(method, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancelParent := context.WithCancel(context.Background())
				defer cancelParent()
				cfg := lifecycle.DefaultConfig()
				cfg.SetEnabled(true)
				pool := NewPool(ctx, cfg)
				defer pool.Shutdown()

				var called atomic.Int32
				cancel := pool.ScheduleOnce("test", time.Second, func() { called.Add(1) })
				synctest.Wait() // The worker is waiting on its timer.
				switch method {
				case "returned":
					cancel()
				case "named":
					pool.Cancel("test")
				case "parent":
					cancelParent()
				}
				time.Sleep(2 * time.Second)
				synctest.Wait()
				if got := called.Load(); got != 0 {
					t.Fatalf("callbacks after cancellation = %d, want 0", got)
				}
			})
		})
	}
}

func TestScheduleInterval_Cancel(t *testing.T) {
	for _, method := range []string{"returned", "named", "parent"} {
		t.Run(method, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancelParent := context.WithCancel(context.Background())
				defer cancelParent()
				cfg := lifecycle.DefaultConfig()
				cfg.SetEnabled(true)
				pool := NewPool(ctx, cfg)
				defer pool.Shutdown()

				var called atomic.Int32
				release := make(chan struct{})
				cancel := pool.ScheduleInterval("test", time.Second, func() {
					called.Add(1)
					<-release
				})
				synctest.Wait()
				time.Sleep(time.Second) // A tick is ready behind the active callback.
				switch method {
				case "returned":
					cancel()
				case "named":
					pool.Cancel("test")
				case "parent":
					cancelParent()
				}
				close(release)
				synctest.Wait()
				time.Sleep(2 * time.Second)
				synctest.Wait()
				if got := called.Load(); got != 1 {
					t.Fatalf("callbacks = %d, want only the already admitted callback", got)
				}
			})
		})
	}
}

func TestScheduleInterval_ReplaceByName(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := lifecycle.DefaultConfig()
		cfg.SetEnabled(true)
		pool := NewPool(context.Background(), cfg)
		defer pool.Shutdown()

		var first, second atomic.Int32
		release := make(chan struct{})
		pool.ScheduleInterval("worker", time.Second, func() {
			first.Add(1)
			<-release
		})
		synctest.Wait()
		time.Sleep(time.Second) // A tick is ready while the old callback is blocked.
		pool.ScheduleInterval("worker", time.Second, func() { second.Add(1) })
		close(release)
		synctest.Wait()
		time.Sleep(2500 * time.Millisecond)
		synctest.Wait()

		if got := first.Load(); got != 1 {
			t.Errorf("replaced worker callbacks = %d, want 1", got)
		}
		if got := second.Load(); got != 3 {
			t.Errorf("replacement callbacks = %d, want initial call and two ticks", got)
		}
	})
}

func TestShutdown_StopsAll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := lifecycle.DefaultConfig()
		cfg.SetEnabled(true)
		pool := NewPool(context.Background(), cfg)
		defer pool.Shutdown()

		var intervals, once atomic.Int32
		release := make(chan struct{})
		for _, name := range []string{"a", "b"} {
			pool.ScheduleInterval(name, time.Second, func() {
				intervals.Add(1)
				<-release
			})
		}
		pool.ScheduleOnce("pending", 2*time.Second, func() { once.Add(1) })
		synctest.Wait()
		time.Sleep(time.Second) // Both callbacks are blocked with a tick ready.
		pool.Shutdown()         // Must return without joining the active callbacks.
		close(release)
		synctest.Wait()
		time.Sleep(3 * time.Second)
		synctest.Wait()

		if got := intervals.Load(); got != 2 {
			t.Errorf("interval callbacks = %d, want only the two already admitted", got)
		}
		if got := once.Load(); got != 0 {
			t.Errorf("pending one-shot callbacks = %d, want 0", got)
		}
	})
}

func TestSchedule_CanceledPool(t *testing.T) {
	for _, stop := range []string{"parent", "shutdown"} {
		for _, mode := range []struct {
			name              string
			interval, enabled bool
			delay             time.Duration
		}{
			{"interval_instant", true, false, time.Second},
			{"interval_delayed", true, true, time.Second},
			{"interval_zero", true, true, 0},
			{"once_instant", false, false, time.Second},
			{"once_zero", false, true, 0},
			{"once_delayed", false, true, time.Second},
		} {
			t.Run(stop+"/"+mode.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					if stop == "parent" {
						cancel()
					}
					cfg := lifecycle.DefaultConfig()
					cfg.SetEnabled(mode.enabled)
					pool := NewPool(ctx, cfg)
					defer pool.Shutdown()
					if stop == "shutdown" {
						pool.Shutdown()
					}

					var called atomic.Int32
					fn := func() { called.Add(1) }
					var cancelWorker func()
					if mode.interval {
						cancelWorker = pool.ScheduleInterval("test", mode.delay, fn)
					} else {
						cancelWorker = pool.ScheduleOnce("test", mode.delay, fn)
					}
					synctest.Wait()
					time.Sleep(2 * time.Second)
					synctest.Wait()
					cancelWorker() // Rejected schedules still return a usable cancellation function.
					if got := called.Load(); got != 0 {
						t.Fatalf("callbacks after %s cancellation = %d, want 0", stop, got)
					}
				})
			})
		}
	}
}
