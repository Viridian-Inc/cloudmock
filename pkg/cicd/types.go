package cicd

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a pipeline is not found.
var ErrNotFound = errors.New("pipeline not found")

// Pipeline represents a CI/CD pipeline run.
type Pipeline struct {
	ID         string     `json:"id"`
	Provider   string     `json:"provider"`   // "github_actions", "gitlab_ci", "jenkins"
	Repo       string     `json:"repo"`
	Branch     string     `json:"branch"`
	CommitHash string     `json:"commit_hash"`
	Status     string     `json:"status"` // "running", "success", "failure", "cancelled"
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	DurationMs int64      `json:"duration_ms"`
	URL        string     `json:"url"` // link to CI provider
	Jobs       []Job      `json:"jobs"`
}

// Job represents a single job within a pipeline.
type Job struct {
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	DurationMs int64      `json:"duration_ms"`
}

// TestResult represents a single test result within a pipeline run.
type TestResult struct {
	PipelineID string  `json:"pipeline_id"`
	Suite      string  `json:"suite"`
	Name       string  `json:"name"`
	Status     string  `json:"status"` // "passed", "failed", "skipped"
	DurationMs float64 `json:"duration_ms"`
	Error      string  `json:"error,omitempty"`
}

// CISummary provides an overall CI health overview.
type CISummary struct {
	TotalPipelines int            `json:"total_pipelines"`
	PassRate       float64        `json:"pass_rate"`
	AvgDurationMs  float64        `json:"avg_duration_ms"`
	FlakyTests     []FlakyTest    `json:"flaky_tests"`
	ByProvider     map[string]int `json:"by_provider"`
}

// FlakyTest is a test that has mixed pass/fail results.
type FlakyTest struct {
	Suite    string  `json:"suite"`
	Name     string  `json:"name"`
	PassRate float64 `json:"pass_rate"`
	Runs     int     `json:"runs"`
}
