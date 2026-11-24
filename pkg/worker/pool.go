// Package worker provides a background worker pool for services that need
// periodic or deferred work (health checks, capacity reconciliation, event firing).
package worker

import (
	"context"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
)

// Pool manages background goroutines for service work.
type Pool struct {
	ctx     context.Context
	cancel  context.CancelFunc
	lcCfg   *lifecycle.Config
	mu      sync.Mutex
	workers map[string]context.CancelFunc
}

// NewPool creates a worker pool. Background workers stop when ctx is cancelled.
// lcCfg controls timing: when delays are disabled, intervals fire immediately.
func NewPool(ctx context.Context, lcCfg *lifecycle.Config) *Pool {
	ctx, cancel := context.WithCancel(ctx)
	if lcCfg == nil {
		lcCfg = lifecycle.DefaultConfig()
	}
	return &Pool{
		ctx:     ctx,
		cancel:  cancel,
		lcCfg:   lcCfg,
		workers: make(map[string]context.CancelFunc),
	}
}

// ScheduleInterval runs fn repeatedly at the given interval.
// In instant mode (lifecycle delays disabled), fn is called once immediately
// and not repeated. Returns a cancel function.
func (p *Pool) ScheduleInterval(name string, interval time.Duration, fn func()) func() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctx.Err() != nil {
		return func() {}
	}

	// Cancel any existing worker with the same name.
	if cancel, ok := p.workers[name]; ok {
		cancel()
	}

	wCtx, wCancel := context.WithCancel(p.ctx)
	p.workers[name] = wCancel

	go func() {
		// In instant mode, fire once immediately.
		if !p.lcCfg.Enabled() {
			invokeIfActive(wCtx, fn)
			return
		}

		effective := p.lcCfg.EffectiveDelay(interval)
		if effective <= 0 {
			invokeIfActive(wCtx, fn)
			return
		}

		ticker := time.NewTicker(effective)
		defer ticker.Stop()

		// Fire once immediately, then on interval.
		if !invokeIfActive(wCtx, fn) {
			return
		}

		for {
			select {
			case <-wCtx.Done():
				return
			case <-ticker.C:
				if !invokeIfActive(wCtx, fn) {
					return
				}
			}
		}
	}()

	return wCancel
}

// ScheduleOnce runs fn after a delay. In instant mode, fn runs immediately.
// Returns a cancel function.
func (p *Pool) ScheduleOnce(name string, delay time.Duration, fn func()) func() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctx.Err() != nil {
		return func() {}
	}

	if cancel, ok := p.workers[name]; ok {
		cancel()
	}

	wCtx, wCancel := context.WithCancel(p.ctx)
	p.workers[name] = wCancel

	go func() {
		effective := p.lcCfg.EffectiveDelay(delay)
		if effective <= 0 {
			invokeIfActive(wCtx, fn)
			return
		}

		timer := time.NewTimer(effective)
		defer timer.Stop()

		select {
		case <-wCtx.Done():
			return
		case <-timer.C:
			invokeIfActive(wCtx, fn)
		}
	}()

	return wCancel
}

// invokeIfActive admits a callback only while its worker context is active.
// Cancellation does not wait for a callback already admitted to finish.
func invokeIfActive(ctx context.Context, fn func()) bool {
	if ctx.Err() != nil {
		return false
	}
	fn()
	return true
}

// Cancel stops a specific named worker.
func (p *Pool) Cancel(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cancel, ok := p.workers[name]; ok {
		cancel()
		delete(p.workers, name)
	}
}

// Shutdown cancels all workers and prevents new ones from being scheduled.
// It does not wait for callbacks already admitted for execution to finish.
func (p *Pool) Shutdown() {
	p.cancel()
}
