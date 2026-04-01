package traffic

import (
	"time"
)

// RecordingStatus describes the state of a traffic recording.
type RecordingStatus string

const (
	RecordingActive    RecordingStatus = "active"
	RecordingCompleted RecordingStatus = "completed"
	RecordingStopped   RecordingStatus = "stopped"
)

// ReplayStatus describes the state of a replay run.
type ReplayStatus string

const (
	ReplayPending   ReplayStatus = "pending"
	ReplayRunning   ReplayStatus = "running"
	ReplayPaused    ReplayStatus = "paused"
	ReplayCompleted ReplayStatus = "completed"
	ReplayCancelled ReplayStatus = "cancelled"
	ReplayFailed    ReplayStatus = "failed"
)

// CapturedEntry is a single request captured during a recording session.
type CapturedEntry struct {
	ID             string            `json:"id"`
	Timestamp      time.Time         `json:"timestamp"`
	Service        string            `json:"service"`
	Action         string            `json:"action"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	StatusCode     int               `json:"status_code"`
	LatencyMs      float64           `json:"latency_ms"`
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	RequestBody    string            `json:"request_body,omitempty"`
	ResponseBody   string            `json:"response_body,omitempty"`
	OffsetMs       float64           `json:"offset_ms"` // milliseconds since recording start
}

// Recording holds a named set of captured traffic.
type Recording struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Status      RecordingStatus `json:"status"`
	Filter      RecordingFilter `json:"filter"`
	DurationSec int             `json:"duration_sec"`
	StartedAt   time.Time       `json:"started_at"`
	StoppedAt   *time.Time      `json:"stopped_at,omitempty"`
	EntryCount  int             `json:"entry_count"`
	Entries     []CapturedEntry `json:"entries,omitempty"`
}

// RecordingFilter constrains which requests are captured.
type RecordingFilter struct {
	Service string `json:"service,omitempty"`
	Path    string `json:"path,omitempty"`
	Method  string `json:"method,omitempty"`
}

// ReplayRun tracks a single replay execution.
type ReplayRun struct {
	ID          string       `json:"id"`
	RecordingID string       `json:"recording_id"`
	Status      ReplayStatus `json:"status"`
	Speed       float64      `json:"speed"` // 1.0 = realtime, 2.0 = 2x, etc.
	StartedAt   time.Time    `json:"started_at"`
	FinishedAt  *time.Time   `json:"finished_at,omitempty"`
	TotalCount  int          `json:"total_count"`
	ReplayedCount int        `json:"replayed_count"`
	MatchCount  int          `json:"match_count"`
	MismatchCount int        `json:"mismatch_count"`
	ErrorCount  int          `json:"error_count"`
	Results     []ReplayResult `json:"results,omitempty"`
	Stats       LatencyStats `json:"stats"`
}

// ReplayResult captures the outcome of replaying one captured entry.
type ReplayResult struct {
	EntryID        string  `json:"entry_id"`
	OriginalStatus int     `json:"original_status"`
	OriginalMs     float64 `json:"original_latency_ms"`
	ReplayStatus   int     `json:"replay_status"`
	ReplayMs       float64 `json:"replay_latency_ms"`
	Match          bool    `json:"match"`
	LatencyDelta   float64 `json:"latency_delta_ms"`
	Error          string  `json:"error,omitempty"`
}

// LatencyStats summarises latency across a replay run.
type LatencyStats struct {
	MinMs float64 `json:"min_ms"`
	MaxMs float64 `json:"max_ms"`
	AvgMs float64 `json:"avg_ms"`
	P50Ms float64 `json:"p50_ms"`
	P95Ms float64 `json:"p95_ms"`
	P99Ms float64 `json:"p99_ms"`
}

// SyntheticScenario defines a template for generating synthetic traffic recordings.
type SyntheticScenario struct {
	Name       string          `json:"name"`
	Service    string          `json:"service"`
	Action     string          `json:"action"`
	Method     string          `json:"method"`
	Path       string          `json:"path"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string          `json:"body,omitempty"`
	Count      int             `json:"count"`      // number of entries to generate
	IntervalMs int             `json:"interval_ms"` // spacing between entries
}

// ReplaySchedule defines a cron-like schedule for running replays automatically.
type ReplaySchedule struct {
	ID          string  `json:"id"`
	RecordingID string  `json:"recording_id"`
	Speed       float64 `json:"speed"`
	CronExpr    string  `json:"cron_expr"`
	Enabled     bool    `json:"enabled"`
}

// RunComparison holds a side-by-side comparison of two replay runs.
type RunComparison struct {
	RunA         RunSummary   `json:"run_a"`
	RunB         RunSummary   `json:"run_b"`
	LatencyDelta LatencyStats `json:"latency_delta"`
	MatchDelta   float64      `json:"match_rate_delta"` // runB match% - runA match%
}

// RunSummary is a compact summary of a replay run for comparisons.
type RunSummary struct {
	ID            string       `json:"id"`
	RecordingID   string       `json:"recording_id"`
	Status        ReplayStatus `json:"status"`
	TotalCount    int          `json:"total_count"`
	MatchRate     float64      `json:"match_rate"`
	Stats         LatencyStats `json:"stats"`
}
