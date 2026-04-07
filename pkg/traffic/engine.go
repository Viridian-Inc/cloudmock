package traffic

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

// Broadcaster is the subset of admin.EventBroadcaster the engine needs.
type Broadcaster interface {
	Broadcast(eventType string, data any)
}

// Engine orchestrates traffic recording, replay, and synthetic generation.
type Engine struct {
	store       RecordingStore
	requestLog  *gateway.RequestLog
	gwPort      int
	broadcaster Broadcaster

	mu             sync.Mutex
	activeRecID    string
	activeRecStop  context.CancelFunc
	activeReplays  map[string]context.CancelFunc
	pausedReplays  map[string]chan struct{} // signalled to resume
}

// New creates a traffic engine.
func New(store RecordingStore, log *gateway.RequestLog, gwPort int) *Engine {
	return &Engine{
		store:         store,
		requestLog:    log,
		gwPort:        gwPort,
		activeReplays: make(map[string]context.CancelFunc),
		pausedReplays: make(map[string]chan struct{}),
	}
}

// SetBroadcaster wires an SSE broadcaster for progress events.
func (e *Engine) SetBroadcaster(b Broadcaster) {
	e.broadcaster = b
}

// Store returns the underlying RecordingStore.
func (e *Engine) Store() RecordingStore {
	return e.store
}

// ---------- Recording ----------

// StartRecording begins capturing live traffic from the RequestLog.
// It polls the log every 250ms for new entries matching the filter and
// automatically stops after durationSec seconds (0 = indefinite until StopRecording).
func (e *Engine) StartRecording(ctx context.Context, name string, durationSec int, filter RecordingFilter) (*Recording, error) {
	e.mu.Lock()
	if e.activeRecID != "" {
		e.mu.Unlock()
		return nil, fmt.Errorf("recording %s already active", e.activeRecID)
	}

	rec := &Recording{
		Name:        name,
		Status:      RecordingActive,
		Filter:      filter,
		DurationSec: durationSec,
		StartedAt:   time.Now(), // Use local time to match RequestLog entry timestamps
	}
	if err := e.store.SaveRecording(ctx, rec); err != nil {
		e.mu.Unlock()
		return nil, err
	}

	recCtx, cancel := context.WithCancel(ctx)
	e.activeRecID = rec.ID
	e.activeRecStop = cancel
	e.mu.Unlock()

	go e.captureLoop(recCtx, rec.ID, rec.StartedAt, durationSec, filter)

	return rec, nil
}

// StopRecording stops the currently active recording.
func (e *Engine) StopRecording(ctx context.Context) (*Recording, error) {
	e.mu.Lock()
	id := e.activeRecID
	cancel := e.activeRecStop
	e.activeRecID = ""
	e.activeRecStop = nil
	e.mu.Unlock()

	if id == "" {
		return nil, fmt.Errorf("no active recording")
	}
	cancel()

	// Give the capture loop a moment to flush, then update status.
	time.Sleep(50 * time.Millisecond)

	rec, err := e.store.GetRecording(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rec.Status = RecordingStopped
	rec.StoppedAt = &now
	if err := e.store.SaveRecording(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// captureLoop polls the RequestLog for new entries until the context is cancelled
// or the duration elapses.
func (e *Engine) captureLoop(ctx context.Context, recID string, start time.Time, durationSec int, filter RecordingFilter) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var deadline <-chan time.Time
	if durationSec > 0 {
		timer := time.NewTimer(time.Duration(durationSec) * time.Second)
		defer timer.Stop()
		deadline = timer.C
	}

	seen := make(map[string]struct{})

	for {
		select {
		case <-ctx.Done():
			e.finishRecording(recID)
			return
		case <-deadline:
			e.finishRecording(recID)
			e.mu.Lock()
			if e.activeRecID == recID {
				e.activeRecID = ""
				e.activeRecStop = nil
			}
			e.mu.Unlock()
			return
		case <-ticker.C:
			entries := e.requestLog.RecentFiltered(gateway.RequestFilter{
				Service: filter.Service,
				Path:    filter.Path,
				Method:  filter.Method,
				Limit:   200,
			})

			var newEntries []CapturedEntry
			for _, ent := range entries {
				if _, ok := seen[ent.ID]; ok {
					continue
				}
				if ent.Timestamp.Before(start) {
					continue
				}
				// Skip replay requests to avoid recursive capture.
				if ent.RequestHeaders["X-Cloudmock-Replay"] != "" {
					continue
				}
				seen[ent.ID] = struct{}{}
				newEntries = append(newEntries, CapturedEntry{
					ID:             ent.ID,
					Timestamp:      ent.Timestamp,
					Service:        ent.Service,
					Action:         ent.Action,
					Method:         ent.Method,
					Path:           ent.Path,
					StatusCode:     ent.StatusCode,
					LatencyMs:      ent.LatencyMs,
					RequestHeaders: ent.RequestHeaders,
					RequestBody:    ent.RequestBody,
					ResponseBody:   ent.ResponseBody,
					OffsetMs:       float64(ent.Timestamp.Sub(start).Milliseconds()),
				})
			}

			if len(newEntries) > 0 {
				rec, err := e.store.GetRecording(ctx, recID)
				if err != nil {
					return
				}
				rec.Entries = append(rec.Entries, newEntries...)
				rec.EntryCount = len(rec.Entries)
				_ = e.store.SaveRecording(ctx, rec)

				if e.broadcaster != nil {
					e.broadcaster.Broadcast("traffic_capture", map[string]any{
						"recording_id": recID,
						"new_count":    len(newEntries),
						"total_count":  rec.EntryCount,
					})
				}
			}
		}
	}
}

func (e *Engine) finishRecording(recID string) {
	ctx := context.Background()
	rec, err := e.store.GetRecording(ctx, recID)
	if err != nil {
		return
	}
	if rec.Status == RecordingActive {
		now := time.Now().UTC()
		rec.Status = RecordingCompleted
		rec.StoppedAt = &now
		_ = e.store.SaveRecording(ctx, rec)
	}
}

// ---------- Replay ----------

// StartReplay replays a recording against the gateway at the given speed multiplier.
func (e *Engine) StartReplay(ctx context.Context, recordingID string, speed float64) (*ReplayRun, error) {
	rec, err := e.store.GetRecording(ctx, recordingID)
	if err != nil {
		return nil, fmt.Errorf("recording not found: %w", err)
	}
	if len(rec.Entries) == 0 {
		return nil, fmt.Errorf("recording %s has no entries", recordingID)
	}
	if speed <= 0 {
		speed = 1.0
	}

	run := &ReplayRun{
		RecordingID: recordingID,
		Status:      ReplayRunning,
		Speed:       speed,
		TotalCount:  len(rec.Entries),
		StartedAt:   time.Now().UTC(),
	}
	if err := e.store.SaveRun(ctx, run); err != nil {
		return nil, err
	}

	replayCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.activeReplays[run.ID] = cancel
	e.mu.Unlock()

	go e.replayLoop(replayCtx, run.ID, rec.Entries, speed)

	return run, nil
}

// PauseReplay pauses a running replay.
func (e *Engine) PauseReplay(ctx context.Context, runID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.activeReplays[runID]; !ok {
		return fmt.Errorf("replay %s not active", runID)
	}
	if _, ok := e.pausedReplays[runID]; ok {
		return fmt.Errorf("replay %s already paused", runID)
	}

	e.pausedReplays[runID] = make(chan struct{})

	run, err := e.store.GetRun(ctx, runID)
	if err == nil {
		run.Status = ReplayPaused
		_ = e.store.UpdateRun(ctx, run)
	}
	return nil
}

// ResumeReplay resumes a paused replay.
func (e *Engine) ResumeReplay(ctx context.Context, runID string) error {
	e.mu.Lock()
	ch, ok := e.pausedReplays[runID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("replay %s not paused", runID)
	}
	delete(e.pausedReplays, runID)
	e.mu.Unlock()

	close(ch) // unblock the replay loop

	run, err := e.store.GetRun(ctx, runID)
	if err == nil {
		run.Status = ReplayRunning
		_ = e.store.UpdateRun(ctx, run)
	}
	return nil
}

// CancelReplay cancels a running or paused replay.
func (e *Engine) CancelReplay(ctx context.Context, runID string) error {
	e.mu.Lock()
	cancel, ok := e.activeReplays[runID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("replay %s not active", runID)
	}
	delete(e.activeReplays, runID)
	// Also unblock if paused.
	if ch, paused := e.pausedReplays[runID]; paused {
		delete(e.pausedReplays, runID)
		close(ch)
	}
	e.mu.Unlock()

	cancel()

	run, err := e.store.GetRun(ctx, runID)
	if err == nil {
		now := time.Now().UTC()
		run.Status = ReplayCancelled
		run.FinishedAt = &now
		_ = e.store.UpdateRun(ctx, run)
	}
	return nil
}

func (e *Engine) replayLoop(ctx context.Context, runID string, entries []CapturedEntry, speed float64) {
	// Sort entries by offset so we replay in chronological order.
	sorted := make([]CapturedEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OffsetMs < sorted[j].OffsetMs
	})

	var allResults []ReplayResult
	var matchCount, mismatchCount, errorCount int

	client := &http.Client{Timeout: 30 * time.Second}

	for i, entry := range sorted {
		// Check cancellation.
		select {
		case <-ctx.Done():
			e.finishReplay(runID, allResults, matchCount, mismatchCount, errorCount, ReplayCancelled)
			return
		default:
		}

		// Check if paused.
		e.mu.Lock()
		pauseCh, paused := e.pausedReplays[runID]
		e.mu.Unlock()
		if paused {
			select {
			case <-pauseCh:
				// resumed
			case <-ctx.Done():
				e.finishReplay(runID, allResults, matchCount, mismatchCount, errorCount, ReplayCancelled)
				return
			}
		}

		// Wait for the correct offset relative to speed.
		if i > 0 {
			delta := sorted[i].OffsetMs - sorted[i-1].OffsetMs
			if delta > 0 && speed > 0 {
				waitMs := delta / speed
				select {
				case <-time.After(time.Duration(waitMs) * time.Millisecond):
				case <-ctx.Done():
					e.finishReplay(runID, allResults, matchCount, mismatchCount, errorCount, ReplayCancelled)
					return
				}
			}
		}

		result := e.replayEntry(client, entry)
		allResults = append(allResults, result)

		if result.Error != "" {
			errorCount++
		} else if result.Match {
			matchCount++
		} else {
			mismatchCount++
		}

		// Periodically persist progress and broadcast.
		if (i+1)%10 == 0 || i == len(sorted)-1 {
			run, err := e.store.GetRun(ctx, runID)
			if err == nil {
				run.ReplayedCount = i + 1
				run.MatchCount = matchCount
				run.MismatchCount = mismatchCount
				run.ErrorCount = errorCount
				run.Results = allResults
				run.Stats = computeLatencyStats(allResults)
				_ = e.store.UpdateRun(ctx, run)
			}

			if e.broadcaster != nil {
				e.broadcaster.Broadcast("traffic_replay", map[string]any{
					"run_id":        runID,
					"replayed":      i + 1,
					"total":         len(sorted),
					"match_count":   matchCount,
					"mismatch_count": mismatchCount,
					"error_count":   errorCount,
				})
			}
		}
	}

	e.finishReplay(runID, allResults, matchCount, mismatchCount, errorCount, ReplayCompleted)
}

func (e *Engine) finishReplay(runID string, results []ReplayResult, matchCount, mismatchCount, errorCount int, status ReplayStatus) {
	ctx := context.Background()
	run, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	run.Status = status
	run.FinishedAt = &now
	run.ReplayedCount = len(results)
	run.MatchCount = matchCount
	run.MismatchCount = mismatchCount
	run.ErrorCount = errorCount
	run.Results = results
	run.Stats = computeLatencyStats(results)
	_ = e.store.UpdateRun(ctx, run)

	e.mu.Lock()
	delete(e.activeReplays, runID)
	delete(e.pausedReplays, runID)
	e.mu.Unlock()

	if e.broadcaster != nil {
		e.broadcaster.Broadcast("traffic_replay_done", map[string]any{
			"run_id": runID,
			"status": string(status),
			"total":  len(results),
			"match":  matchCount,
		})
	}
}

func (e *Engine) replayEntry(client *http.Client, entry CapturedEntry) ReplayResult {
	gwURL := fmt.Sprintf("http://localhost:%d%s", e.gwPort, entry.Path)

	var body io.Reader
	if entry.RequestBody != "" {
		body = strings.NewReader(entry.RequestBody)
	}

	req, err := http.NewRequest(entry.Method, gwURL, body)
	if err != nil {
		return ReplayResult{
			EntryID:        entry.ID,
			OriginalStatus: entry.StatusCode,
			OriginalMs:     entry.LatencyMs,
			Error:          "failed to create request: " + err.Error(),
		}
	}

	for k, v := range entry.RequestHeaders {
		req.Header.Set(k, v)
	}
	// Mark as replay so logging middleware can identify it.
	req.Header.Set("X-Cloudmock-Replay", entry.ID)

	start := time.Now()
	resp, err := client.Do(req)
	replayMs := float64(time.Since(start).Nanoseconds()) / 1e6
	if err != nil {
		return ReplayResult{
			EntryID:        entry.ID,
			OriginalStatus: entry.StatusCode,
			OriginalMs:     entry.LatencyMs,
			ReplayMs:       replayMs,
			Error:          "request failed: " + err.Error(),
		}
	}
	defer resp.Body.Close()
	// Drain body so connection can be reused.
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 10240))

	return ReplayResult{
		EntryID:        entry.ID,
		OriginalStatus: entry.StatusCode,
		OriginalMs:     entry.LatencyMs,
		ReplayStatus:   resp.StatusCode,
		ReplayMs:       replayMs,
		Match:          resp.StatusCode == entry.StatusCode,
		LatencyDelta:   replayMs - entry.LatencyMs,
	}
}

// ---------- Synthetic ----------

// GenerateSynthetic creates a recording from a template scenario.
func (e *Engine) GenerateSynthetic(ctx context.Context, scenario SyntheticScenario) (*Recording, error) {
	if scenario.Count <= 0 {
		scenario.Count = 10
	}
	if scenario.IntervalMs <= 0 {
		scenario.IntervalMs = 100
	}
	if scenario.Method == "" {
		scenario.Method = "POST"
	}
	if scenario.Name == "" {
		scenario.Name = fmt.Sprintf("synthetic-%s-%s", scenario.Service, scenario.Action)
	}

	now := time.Now().UTC()
	entries := make([]CapturedEntry, scenario.Count)
	for i := 0; i < scenario.Count; i++ {
		offsetMs := float64(i * scenario.IntervalMs)
		entries[i] = CapturedEntry{
			ID:             fmt.Sprintf("syn-%d", i+1),
			Timestamp:      now.Add(time.Duration(offsetMs) * time.Millisecond),
			Service:        scenario.Service,
			Action:         scenario.Action,
			Method:         scenario.Method,
			Path:           scenario.Path,
			StatusCode:     200,
			LatencyMs:      0,
			RequestHeaders: scenario.Headers,
			RequestBody:    scenario.Body,
			OffsetMs:       offsetMs,
		}
	}

	rec := &Recording{
		Name:        scenario.Name,
		Status:      RecordingCompleted,
		DurationSec: int(float64(scenario.Count*scenario.IntervalMs) / 1000),
		StartedAt:   now,
		EntryCount:  len(entries),
		Entries:     entries,
	}
	stopped := now
	rec.StoppedAt = &stopped

	if err := e.store.SaveRecording(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// ---------- Compare ----------

// CompareRuns produces a side-by-side comparison of two replay runs.
func (e *Engine) CompareRuns(ctx context.Context, runAID, runBID string) (*RunComparison, error) {
	runA, err := e.store.GetRun(ctx, runAID)
	if err != nil {
		return nil, fmt.Errorf("run A: %w", err)
	}
	runB, err := e.store.GetRun(ctx, runBID)
	if err != nil {
		return nil, fmt.Errorf("run B: %w", err)
	}

	matchRateA := 0.0
	if runA.TotalCount > 0 {
		matchRateA = float64(runA.MatchCount) / float64(runA.TotalCount) * 100
	}
	matchRateB := 0.0
	if runB.TotalCount > 0 {
		matchRateB = float64(runB.MatchCount) / float64(runB.TotalCount) * 100
	}

	return &RunComparison{
		RunA: RunSummary{
			ID:          runA.ID,
			RecordingID: runA.RecordingID,
			Status:      runA.Status,
			TotalCount:  runA.TotalCount,
			MatchRate:   matchRateA,
			Stats:       runA.Stats,
		},
		RunB: RunSummary{
			ID:          runB.ID,
			RecordingID: runB.RecordingID,
			Status:      runB.Status,
			TotalCount:  runB.TotalCount,
			MatchRate:   matchRateB,
			Stats:       runB.Stats,
		},
		LatencyDelta: LatencyStats{
			MinMs: runB.Stats.MinMs - runA.Stats.MinMs,
			MaxMs: runB.Stats.MaxMs - runA.Stats.MaxMs,
			AvgMs: runB.Stats.AvgMs - runA.Stats.AvgMs,
			P50Ms: runB.Stats.P50Ms - runA.Stats.P50Ms,
			P95Ms: runB.Stats.P95Ms - runA.Stats.P95Ms,
			P99Ms: runB.Stats.P99Ms - runA.Stats.P99Ms,
		},
		MatchDelta: matchRateB - matchRateA,
	}, nil
}

// ---------- Helpers ----------

func computeLatencyStats(results []ReplayResult) LatencyStats {
	if len(results) == 0 {
		return LatencyStats{}
	}

	var latencies []float64
	for _, r := range results {
		if r.ReplayMs > 0 {
			latencies = append(latencies, r.ReplayMs)
		}
	}
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	sort.Float64s(latencies)
	n := len(latencies)

	sum := 0.0
	for _, v := range latencies {
		sum += v
	}

	return LatencyStats{
		MinMs: latencies[0],
		MaxMs: latencies[n-1],
		AvgMs: math.Round(sum/float64(n)*100) / 100,
		P50Ms: percentile(latencies, 50),
		P95Ms: percentile(latencies, 95),
		P99Ms: percentile(latencies, 99),
	}
}

func percentile(sorted []float64, pct float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(pct/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
