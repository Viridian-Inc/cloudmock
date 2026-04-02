package harness

import "context"

type Grade string

const (
	GradePass        Grade = "pass"
	GradePartial     Grade = "partial"
	GradeFail        Grade = "fail"
	GradeUnsupported Grade = "unsupported"
)

type LatencyStats struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Mean float64 `json:"mean"`
	P50  float64 `json:"p50"`
	P95  float64 `json:"p95"`
	P99  float64 `json:"p99"`
}

type LoadStats struct {
	LatencyStats
	ThroughputOpsPerSec float64 `json:"throughput_ops_sec"`
}

type OperationResult struct {
	Name        string       `json:"name"`
	ColdMs      float64      `json:"cold_ms"`
	Warm        LatencyStats `json:"warm"`
	Load        LoadStats    `json:"load"`
	Correctness Grade        `json:"correctness"`
	Findings    []Finding    `json:"findings,omitempty"`
}

type Finding struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Grade    Grade  `json:"grade"`
}

type ServiceResult struct {
	Service    string                      `json:"service"`
	Tier       int                         `json:"tier"`
	Operations map[string]*OperationResult `json:"operations"`
}

type TargetResults struct {
	Target   string                    `json:"target"`
	Mode     string                    `json:"mode"`
	Services map[string]*ServiceResult `json:"services"`
}

type StartupResult struct {
	MedianMs float64   `json:"median_ms"`
	Runs     []float64 `json:"runs"`
}

type ResourceStats struct {
	PeakMemoryMB float64 `json:"peak_memory_mb"`
	AvgMemoryMB  float64 `json:"avg_memory_mb"`
	PeakCPUPct   float64 `json:"peak_cpu_pct"`
	AvgCPUPct    float64 `json:"avg_cpu_pct"`
}

type BenchmarkResults struct {
	Meta      Meta                      `json:"meta"`
	Startup   map[string]*StartupResult `json:"startup"`
	Resources map[string]*ResourceStats `json:"resources"`
	Targets   map[string]*TargetResults `json:"targets"`
}

type Meta struct {
	Date              string `json:"date"`
	Platform          string `json:"platform"`
	GoVersion         string `json:"go_version"`
	CloudMockVersion  string `json:"cloudmock_version"`
	LocalStackVersion string `json:"localstack_version"`
	Mode              string `json:"mode"`
	Iterations        int    `json:"iterations"`
	Concurrency       int    `json:"concurrency"`
}

type Suite interface {
	Name() string
	Tier() int
	Operations() []Operation
}

type Operation struct {
	Name     string
	Setup    func(ctx context.Context, endpoint string) error
	Run      func(ctx context.Context, endpoint string) (any, error)
	Teardown func(ctx context.Context, endpoint string) error
	Validate func(resp any) []Finding
}

type Config struct {
	Targets     []string
	Modes       []string
	Services    []string
	Tier        int
	Iterations  int
	Concurrency int
	CI          bool
	Quick       bool
	OutputDir   string
}
