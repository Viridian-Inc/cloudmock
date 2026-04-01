package uptime

import "time"

// Check defines an endpoint to monitor.
type Check struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Method         string            `json:"method"`           // GET, POST, etc.
	ExpectedStatus int               `json:"expected_status"`  // 200, 201, etc.
	Interval       time.Duration     `json:"interval"`         // check interval
	Timeout        time.Duration     `json:"timeout"`          // request timeout
	Headers        map[string]string `json:"headers"`
	Enabled        bool              `json:"enabled"`
}

// CheckResult records the outcome of a single check execution.
type CheckResult struct {
	CheckID    string    `json:"check_id"`
	Timestamp  time.Time `json:"timestamp"`
	StatusCode int       `json:"status_code"`
	ResponseMs float64  `json:"response_ms"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}

// CheckSummary provides an overview of a check's status and history.
type CheckSummary struct {
	Check         Check        `json:"check"`
	Uptime24h     float64      `json:"uptime_24h"`      // percentage
	Uptime7d      float64      `json:"uptime_7d"`
	Uptime30d     float64      `json:"uptime_30d"`
	AvgResponseMs float64     `json:"avg_response_ms"`
	LastResult    *CheckResult `json:"last_result,omitempty"`
	LastFailure   *CheckResult `json:"last_failure,omitempty"`
}
