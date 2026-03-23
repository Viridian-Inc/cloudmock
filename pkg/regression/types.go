package regression

import (
	"time"
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

type AlgorithmType string

const (
	AlgoLatencyRegression AlgorithmType = "latency_regression"
	AlgoErrorRate         AlgorithmType = "error_rate"
	AlgoTenantOutlier     AlgorithmType = "tenant_outlier"
	AlgoCacheMiss         AlgorithmType = "cache_miss"
	AlgoDBFanout          AlgorithmType = "db_fanout"
	AlgoPayloadGrowth     AlgorithmType = "payload_growth"
)

type Regression struct {
	ID            string
	Algorithm     AlgorithmType
	Severity      Severity
	Confidence    int
	Service       string
	Action        string
	DeployID      string
	TenantID      string
	Title         string
	BeforeValue   float64
	AfterValue    float64
	ChangePercent float64
	SampleSize    int64
	DetectedAt    time.Time
	WindowBefore  TimeWindow
	WindowAfter   TimeWindow
	Status        string // "active", "resolved", "dismissed"
	ResolvedAt    *time.Time
}

type TimeWindow struct {
	Start time.Time
	End   time.Time
}

type WindowMetrics struct {
	Service       string
	Action        string
	P50Ms         float64
	P95Ms         float64
	P99Ms         float64
	ErrorRate     float64
	RequestCount  int64
	CacheMissRate float64
	AvgSpanCount  float64
	AvgRespSize   float64
}

// Algorithm configs
type AlgorithmConfig struct {
	LatencyRegression LatencyConfig   `yaml:"latency_regression" json:"latency_regression"`
	ErrorRate         ErrorConfig     `yaml:"error_rate" json:"error_rate"`
	TenantOutlier     OutlierConfig   `yaml:"tenant_outlier" json:"tenant_outlier"`
	CacheMiss         CacheMissConfig `yaml:"cache_miss" json:"cache_miss"`
	DBFanout          FanoutConfig    `yaml:"db_fanout" json:"db_fanout"`
	PayloadGrowth     PayloadConfig   `yaml:"payload_growth" json:"payload_growth"`
}

type LatencyConfig struct {
	P99ThresholdPercent float64 `yaml:"p99_threshold_percent" json:"p99_threshold_percent"`
	MinSampleSize       int     `yaml:"min_sample_size" json:"min_sample_size"`
}

type ErrorConfig struct {
	ThresholdPP   float64 `yaml:"threshold_pp" json:"threshold_pp"`
	MinSampleSize int     `yaml:"min_sample_size" json:"min_sample_size"`
}

type OutlierConfig struct {
	Multiplier    float64 `yaml:"multiplier" json:"multiplier"`
	MinSampleSize int     `yaml:"min_sample_size" json:"min_sample_size"`
	MaxTenants    int     `yaml:"max_tenants" json:"max_tenants"`
}

type CacheMissConfig struct {
	ThresholdPP   float64 `yaml:"threshold_pp" json:"threshold_pp"`
	MinSampleSize int     `yaml:"min_sample_size" json:"min_sample_size"`
}

type FanoutConfig struct {
	ThresholdPercent float64 `yaml:"threshold_percent" json:"threshold_percent"`
	MinSampleSize    int     `yaml:"min_sample_size" json:"min_sample_size"`
}

type PayloadConfig struct {
	ThresholdPercent float64 `yaml:"threshold_percent" json:"threshold_percent"`
	MinSampleSize    int     `yaml:"min_sample_size" json:"min_sample_size"`
}

func DefaultAlgorithmConfig() AlgorithmConfig {
	return AlgorithmConfig{
		LatencyRegression: LatencyConfig{P99ThresholdPercent: 50, MinSampleSize: 100},
		ErrorRate:         ErrorConfig{ThresholdPP: 5, MinSampleSize: 50},
		TenantOutlier:     OutlierConfig{Multiplier: 3.0, MinSampleSize: 200, MaxTenants: 100},
		CacheMiss:         CacheMissConfig{ThresholdPP: 20, MinSampleSize: 100},
		DBFanout:          FanoutConfig{ThresholdPercent: 50, MinSampleSize: 50},
		PayloadGrowth:     PayloadConfig{ThresholdPercent: 100, MinSampleSize: 50},
	}
}
