package uptime

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/worker"
)

// FailureCallback is called when a check has N consecutive failures.
type FailureCallback func(check Check, consecutiveFailures int, lastResult CheckResult)

// EngineConfig holds configuration for the uptime engine.
type EngineConfig struct {
	// ConsecutiveFailuresThreshold is the number of consecutive failures
	// before the failure callback is triggered. Default 3.
	ConsecutiveFailuresThreshold int
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		ConsecutiveFailuresThreshold: 3,
	}
}

// Engine runs periodic uptime checks using a worker pool.
type Engine struct {
	store    Store
	pool     *worker.Pool
	cfg      EngineConfig
	client   *http.Client
	mu       sync.RWMutex
	cancels  map[string]func() // check ID → cancel function
	onFail   FailureCallback
}

// NewEngine creates an uptime engine.
func NewEngine(store Store, pool *worker.Pool, cfg EngineConfig) *Engine {
	if cfg.ConsecutiveFailuresThreshold <= 0 {
		cfg.ConsecutiveFailuresThreshold = 3
	}
	return &Engine{
		store:   store,
		pool:    pool,
		cfg:     cfg,
		client:  &http.Client{},
		cancels: make(map[string]func()),
	}
}

// SetFailureCallback sets the callback for consecutive failures.
func (e *Engine) SetFailureCallback(cb FailureCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onFail = cb
}

// Store returns the underlying uptime store.
func (e *Engine) Store() Store {
	return e.store
}

// CreateCheck creates a new check and starts scheduling it if enabled.
func (e *Engine) CreateCheck(check Check) error {
	if check.ID == "" {
		check.ID = generateID()
	}
	if check.Method == "" {
		check.Method = "GET"
	}
	if check.ExpectedStatus == 0 {
		check.ExpectedStatus = 200
	}
	if check.Interval == 0 {
		check.Interval = 60 * time.Second
	}
	if check.Timeout == 0 {
		check.Timeout = 10 * time.Second
	}

	if err := e.store.CreateCheck(check); err != nil {
		return err
	}

	if check.Enabled {
		e.scheduleCheck(check)
	}
	return nil
}

// UpdateCheck updates a check and reschedules it.
func (e *Engine) UpdateCheck(check Check) error {
	if err := e.store.UpdateCheck(check); err != nil {
		return err
	}

	// Cancel existing schedule.
	e.mu.Lock()
	if cancel, ok := e.cancels[check.ID]; ok {
		cancel()
		delete(e.cancels, check.ID)
	}
	e.mu.Unlock()

	if check.Enabled {
		e.scheduleCheck(check)
	}
	return nil
}

// DeleteCheck deletes a check and stops its schedule.
func (e *Engine) DeleteCheck(id string) error {
	e.mu.Lock()
	if cancel, ok := e.cancels[id]; ok {
		cancel()
		delete(e.cancels, id)
	}
	e.mu.Unlock()

	return e.store.DeleteCheck(id)
}

// StartAll schedules all enabled checks. Call this once at startup.
func (e *Engine) StartAll() {
	checks, err := e.store.ListChecks()
	if err != nil {
		return
	}
	for _, c := range checks {
		if c.Enabled {
			e.scheduleCheck(c)
		}
	}
}

// Summary returns summaries for all checks.
func (e *Engine) Summary() ([]CheckSummary, error) {
	checks, err := e.store.ListChecks()
	if err != nil {
		return nil, err
	}

	summaries := make([]CheckSummary, 0, len(checks))
	for _, c := range checks {
		s, err := e.checkSummary(c)
		if err != nil {
			continue
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (e *Engine) checkSummary(check Check) (CheckSummary, error) {
	results, err := e.store.Results(check.ID)
	if err != nil {
		return CheckSummary{}, err
	}

	lastResult, _ := e.store.LastResult(check.ID)
	lastFailure, _ := e.store.LastFailure(check.ID)

	now := time.Now()
	summary := CheckSummary{
		Check:       check,
		Uptime24h:   calcUptime(results, now, 24*time.Hour),
		Uptime7d:    calcUptime(results, now, 7*24*time.Hour),
		Uptime30d:   calcUptime(results, now, 30*24*time.Hour),
		LastResult:  lastResult,
		LastFailure: lastFailure,
	}

	// Compute average response time.
	if len(results) > 0 {
		var total float64
		for _, r := range results {
			total += r.ResponseMs
		}
		summary.AvgResponseMs = total / float64(len(results))
	}

	return summary, nil
}

func calcUptime(results []CheckResult, now time.Time, window time.Duration) float64 {
	cutoff := now.Add(-window)
	var total, success int
	for _, r := range results {
		if r.Timestamp.Before(cutoff) {
			continue
		}
		total++
		if r.Success {
			success++
		}
	}
	if total == 0 {
		return 100.0 // no data = assume up
	}
	return float64(success) / float64(total) * 100.0
}

func (e *Engine) scheduleCheck(check Check) {
	cancel := e.pool.ScheduleInterval(
		fmt.Sprintf("uptime-%s", check.ID),
		check.Interval,
		func() {
			e.executeCheck(check)
		},
	)

	e.mu.Lock()
	e.cancels[check.ID] = cancel
	e.mu.Unlock()
}

func (e *Engine) executeCheck(check Check) {
	result := CheckResult{
		CheckID:   check.ID,
		Timestamp: time.Now(),
	}

	client := &http.Client{Timeout: check.Timeout}

	req, err := http.NewRequestWithContext(context.Background(), check.Method, check.URL, nil)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		e.storeAndNotify(check, result)
		return
	}

	for k, v := range check.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	result.ResponseMs = float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		e.storeAndNotify(check, result)
		return
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == check.ExpectedStatus
	if !result.Success {
		result.Error = fmt.Sprintf("expected status %d, got %d", check.ExpectedStatus, resp.StatusCode)
	}

	e.storeAndNotify(check, result)
}

func (e *Engine) storeAndNotify(check Check, result CheckResult) {
	_ = e.store.AddResult(result)

	if !result.Success {
		failures := e.store.ConsecutiveFailures(check.ID)
		e.mu.RLock()
		cb := e.onFail
		e.mu.RUnlock()
		if cb != nil && failures >= e.cfg.ConsecutiveFailuresThreshold {
			cb(check, failures, result)
		}
	}
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("chk_%x", b)
}
