package regression

import (
	"testing"
)

// --- Helper tests ---

func TestClassifySeverity(t *testing.T) {
	tests := []struct {
		name          string
		changePercent float64
		want          Severity
	}{
		{"no regression at 10%", 10, ""},
		{"no regression at 20%", 20, ""},
		{"info at 21%", 21, SeverityInfo},
		{"info at 50%", 50, SeverityInfo},
		{"warning at 51%", 51, SeverityWarning},
		{"warning at 100%", 100, SeverityWarning},
		{"critical at 101%", 101, SeverityCritical},
		{"critical at 300%", 300, SeverityCritical},
		{"no regression at 0%", 0, ""},
		{"no regression negative", -10, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySeverity(tt.changePercent)
			if got != tt.want {
				t.Errorf("classifySeverity(%v) = %q, want %q", tt.changePercent, got, tt.want)
			}
		})
	}
}

func TestComputeConfidence(t *testing.T) {
	tests := []struct {
		name          string
		sampleSize    int64
		changePercent float64
		threshold     float64
		consistent    bool
		want          int
	}{
		{"small sample base", 30, 60, 50, false, 30},
		{"medium sample base", 100, 60, 50, false, 60},
		{"large sample base", 1000, 60, 50, false, 85},
		{"large change bonus", 1000, 110, 50, false, 95},
		{"consistency bonus", 1000, 60, 50, true, 90},
		{"all bonuses", 1000, 110, 50, true, 100},
		{"cap at 100", 1000, 200, 50, true, 100},
		{"small sample with bonuses", 30, 110, 50, true, 45},
		{"medium sample boundary 50", 50, 60, 50, false, 60},
		{"medium sample boundary 500", 500, 60, 50, false, 60},
		{"large sample boundary 501", 501, 60, 50, false, 85},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeConfidence(tt.sampleSize, tt.changePercent, tt.threshold, tt.consistent)
			if got != tt.want {
				t.Errorf("computeConfidence(%d, %v, %v, %v) = %d, want %d",
					tt.sampleSize, tt.changePercent, tt.threshold, tt.consistent, got, tt.want)
			}
		})
	}
}

// --- Latency Regression tests ---

func TestDetectLatencyRegression(t *testing.T) {
	cfg := LatencyConfig{P99ThresholdPercent: 50, MinSampleSize: 100}

	t.Run("no regression below threshold", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 130, P95Ms: 90, RequestCount: 200}
		r := detectLatencyRegression(before, after, cfg)
		if r != nil {
			t.Errorf("expected nil, got regression with change %.1f%%", r.ChangePercent)
		}
	})

	t.Run("warning at threshold boundary", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 160, P95Ms: 100, RequestCount: 200}
		r := detectLatencyRegression(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression, got nil")
		}
		// 60% change is >50%, so warning
		if r.Severity != SeverityWarning {
			t.Errorf("severity = %q, want %q", r.Severity, SeverityWarning)
		}
		if r.Algorithm != AlgoLatencyRegression {
			t.Errorf("algorithm = %q, want %q", r.Algorithm, AlgoLatencyRegression)
		}
	})

	t.Run("critical at 2x", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 500}
		after := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 310, P95Ms: 200, RequestCount: 500}
		r := detectLatencyRegression(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression, got nil")
		}
		if r.Severity != SeverityCritical {
			t.Errorf("severity = %q, want %q", r.Severity, SeverityCritical)
		}
	})

	t.Run("low confidence with small sample", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 100}
		after := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 200, P95Ms: 150, RequestCount: 100}
		r := detectLatencyRegression(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression, got nil")
		}
		if r.Confidence > 70 {
			t.Errorf("confidence = %d, expected low confidence for small sample", r.Confidence)
		}
	})

	t.Run("skip below min sample size", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 50}
		after := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 300, P95Ms: 200, RequestCount: 50}
		r := detectLatencyRegression(before, after, cfg)
		if r != nil {
			t.Error("expected nil for below min sample size")
		}
	})

	t.Run("consistency bonus when P95 also regressed", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 600}
		after := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 200, P95Ms: 120, RequestCount: 600}
		rConsistent := detectLatencyRegression(before, after, cfg)

		before2 := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 600}
		after2 := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 200, P95Ms: 85, RequestCount: 600}
		rInconsistent := detectLatencyRegression(before2, after2, cfg)

		if rConsistent == nil || rInconsistent == nil {
			t.Fatal("expected both regressions to be non-nil")
		}
		if rConsistent.Confidence <= rInconsistent.Confidence {
			t.Errorf("consistent confidence %d should be > inconsistent %d",
				rConsistent.Confidence, rInconsistent.Confidence)
		}
	})

	t.Run("title format", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, P95Ms: 80, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 200, P95Ms: 120, RequestCount: 200}
		r := detectLatencyRegression(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Service != "api" || r.Action != "GET /users" {
			t.Errorf("service=%q action=%q", r.Service, r.Action)
		}
		if r.BeforeValue != 100 || r.AfterValue != 200 {
			t.Errorf("before=%v after=%v", r.BeforeValue, r.AfterValue)
		}
	})
}

// --- Error Rate tests ---

func TestDetectErrorRate(t *testing.T) {
	cfg := ErrorConfig{ThresholdPP: 5, MinSampleSize: 50}

	t.Run("no regression below threshold", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.01, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.04, RequestCount: 200}
		r := detectErrorRate(before, after, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})

	t.Run("detects error rate spike", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.01, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.10, RequestCount: 200}
		r := detectErrorRate(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Algorithm != AlgoErrorRate {
			t.Errorf("algorithm = %q", r.Algorithm)
		}
	})

	t.Run("skip below min sample", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.01, RequestCount: 20}
		after := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.50, RequestCount: 20}
		r := detectErrorRate(before, after, cfg)
		if r != nil {
			t.Error("expected nil for small sample")
		}
	})

	t.Run("critical severity on massive spike", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.01, RequestCount: 1000}
		after := &WindowMetrics{Service: "api", Action: "GET /users", ErrorRate: 0.50, RequestCount: 1000}
		r := detectErrorRate(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Severity != SeverityCritical {
			t.Errorf("severity = %q, want critical", r.Severity)
		}
	})
}

// --- Tenant Outlier tests ---

func TestDetectTenantOutlier(t *testing.T) {
	cfg := OutlierConfig{Multiplier: 3.0, MinSampleSize: 200}

	t.Run("no outlier within multiplier", func(t *testing.T) {
		tenant := &WindowMetrics{Service: "tenant-123", P99Ms: 250, RequestCount: 300}
		fleet := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, RequestCount: 5000}
		r := detectTenantOutlier(tenant, fleet, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})

	t.Run("detects outlier above multiplier", func(t *testing.T) {
		tenant := &WindowMetrics{Service: "tenant-123", P99Ms: 400, RequestCount: 300}
		fleet := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, RequestCount: 5000}
		r := detectTenantOutlier(tenant, fleet, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Algorithm != AlgoTenantOutlier {
			t.Errorf("algorithm = %q", r.Algorithm)
		}
		if r.Service != "api" {
			t.Errorf("service = %q, want fleet service", r.Service)
		}
	})

	t.Run("skip below min sample", func(t *testing.T) {
		tenant := &WindowMetrics{Service: "tenant-123", P99Ms: 500, RequestCount: 50}
		fleet := &WindowMetrics{Service: "api", Action: "GET /users", P99Ms: 100, RequestCount: 5000}
		r := detectTenantOutlier(tenant, fleet, cfg)
		if r != nil {
			t.Error("expected nil for small sample")
		}
	})
}

// --- Cache Miss tests ---

func TestDetectCacheMiss(t *testing.T) {
	cfg := CacheMissConfig{ThresholdPP: 20, MinSampleSize: 100}

	t.Run("no regression below threshold", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", CacheMissRate: 0.10, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", CacheMissRate: 0.20, RequestCount: 200}
		r := detectCacheMiss(before, after, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})

	t.Run("detects cache miss spike", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", CacheMissRate: 0.10, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", CacheMissRate: 0.50, RequestCount: 200}
		r := detectCacheMiss(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Algorithm != AlgoCacheMiss {
			t.Errorf("algorithm = %q", r.Algorithm)
		}
	})

	t.Run("skip below min sample", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", CacheMissRate: 0.10, RequestCount: 50}
		after := &WindowMetrics{Service: "api", Action: "GET /users", CacheMissRate: 0.90, RequestCount: 50}
		r := detectCacheMiss(before, after, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})
}

// --- DB Fanout tests ---

func TestDetectDBFanout(t *testing.T) {
	cfg := FanoutConfig{ThresholdPercent: 50, MinSampleSize: 50}

	t.Run("no regression below threshold", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 10, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 14, RequestCount: 200}
		r := detectDBFanout(before, after, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})

	t.Run("detects fanout increase", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 10, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 20, RequestCount: 200}
		r := detectDBFanout(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Algorithm != AlgoDBFanout {
			t.Errorf("algorithm = %q", r.Algorithm)
		}
	})

	t.Run("skip below min sample", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 10, RequestCount: 20}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 100, RequestCount: 20}
		r := detectDBFanout(before, after, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})

	t.Run("critical at large increase", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 10, RequestCount: 1000}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgSpanCount: 35, RequestCount: 1000}
		r := detectDBFanout(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Severity != SeverityCritical {
			t.Errorf("severity = %q, want critical", r.Severity)
		}
	})
}

// --- Payload Growth tests ---

func TestDetectPayloadGrowth(t *testing.T) {
	cfg := PayloadConfig{ThresholdPercent: 100, MinSampleSize: 50}

	t.Run("no regression below threshold", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 1000, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 1500, RequestCount: 200}
		r := detectPayloadGrowth(before, after, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})

	t.Run("detects payload growth", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 1000, RequestCount: 200}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 3000, RequestCount: 200}
		r := detectPayloadGrowth(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Algorithm != AlgoPayloadGrowth {
			t.Errorf("algorithm = %q", r.Algorithm)
		}
	})

	t.Run("skip below min sample", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 1000, RequestCount: 20}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 5000, RequestCount: 20}
		r := detectPayloadGrowth(before, after, cfg)
		if r != nil {
			t.Error("expected nil")
		}
	})

	t.Run("critical at massive growth", func(t *testing.T) {
		before := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 1000, RequestCount: 1000}
		after := &WindowMetrics{Service: "api", Action: "GET /users", AvgRespSize: 5000, RequestCount: 1000}
		r := detectPayloadGrowth(before, after, cfg)
		if r == nil {
			t.Fatal("expected regression")
		}
		if r.Severity != SeverityCritical {
			t.Errorf("severity = %q, want critical", r.Severity)
		}
	})
}
