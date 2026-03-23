package regression

import (
	"fmt"
	"math"
)

// classifySeverity maps a change percentage to a severity level.
func classifySeverity(changePercent float64) Severity {
	switch {
	case changePercent > 100:
		return SeverityCritical
	case changePercent > 50:
		return SeverityWarning
	case changePercent > 20:
		return SeverityInfo
	default:
		return ""
	}
}

// computeConfidence produces a 0-100 confidence score based on sample size,
// magnitude of change relative to threshold, and consistency across metrics.
func computeConfidence(sampleSize int64, changePercent, threshold float64, consistent bool) int {
	var base int
	switch {
	case sampleSize > 500:
		base = 85
	case sampleSize >= 50:
		base = 60
	default:
		base = 30
	}

	if changePercent > 2*threshold {
		base += 10
	}
	if consistent {
		base += 5
	}
	if base > 100 {
		base = 100
	}
	return base
}

// detectLatencyRegression checks if P99 latency increased beyond the configured threshold.
func detectLatencyRegression(before, after *WindowMetrics, cfg LatencyConfig) *Regression {
	sampleSize := minInt64(before.RequestCount, after.RequestCount)
	if sampleSize < int64(cfg.MinSampleSize) {
		return nil
	}

	if before.P99Ms == 0 {
		return nil
	}

	changePct := (after.P99Ms - before.P99Ms) / before.P99Ms * 100
	if changePct <= cfg.P99ThresholdPercent {
		return nil
	}

	sev := classifySeverity(changePct)
	if sev == "" {
		return nil
	}

	// Consistency: P95 also regressed >20%
	p95Change := 0.0
	if before.P95Ms > 0 {
		p95Change = (after.P95Ms - before.P95Ms) / before.P95Ms * 100
	}
	consistent := p95Change > 20

	conf := computeConfidence(sampleSize, changePct, cfg.P99ThresholdPercent, consistent)

	return &Regression{
		Algorithm:     AlgoLatencyRegression,
		Severity:      sev,
		Confidence:    conf,
		Service:       before.Service,
		Action:        before.Action,
		Title:         fmt.Sprintf("P99 latency increased %.0f%% (%s)", changePct, before.Service),
		BeforeValue:   before.P99Ms,
		AfterValue:    after.P99Ms,
		ChangePercent: changePct,
		SampleSize:    sampleSize,
		Status:        "active",
	}
}

// detectErrorRate checks if the error rate increased beyond the configured threshold in percentage points.
func detectErrorRate(before, after *WindowMetrics, cfg ErrorConfig) *Regression {
	sampleSize := minInt64(before.RequestCount, after.RequestCount)
	if sampleSize < int64(cfg.MinSampleSize) {
		return nil
	}

	diffPP := (after.ErrorRate - before.ErrorRate) * 100 // convert to percentage points
	if diffPP <= cfg.ThresholdPP {
		return nil
	}

	baseRate := math.Max(before.ErrorRate, 0.001)
	changePct := (after.ErrorRate - before.ErrorRate) / baseRate * 100

	sev := classifySeverity(changePct)
	if sev == "" {
		return nil
	}

	conf := computeConfidence(sampleSize, changePct, cfg.ThresholdPP, false)

	return &Regression{
		Algorithm:     AlgoErrorRate,
		Severity:      sev,
		Confidence:    conf,
		Service:       before.Service,
		Action:        before.Action,
		Title:         fmt.Sprintf("Error rate increased %.1f pp (%s)", diffPP, before.Service),
		BeforeValue:   before.ErrorRate,
		AfterValue:    after.ErrorRate,
		ChangePercent: changePct,
		SampleSize:    sampleSize,
		Status:        "active",
	}
}

// detectTenantOutlier checks if a tenant's P99 latency exceeds the fleet average by the configured multiplier.
func detectTenantOutlier(tenant, fleet *WindowMetrics, cfg OutlierConfig) *Regression {
	if tenant.RequestCount < int64(cfg.MinSampleSize) {
		return nil
	}

	if fleet.P99Ms == 0 {
		return nil
	}

	if tenant.P99Ms <= fleet.P99Ms*cfg.Multiplier {
		return nil
	}

	changePct := (tenant.P99Ms/fleet.P99Ms - 1) * 100

	sev := classifySeverity(changePct)
	if sev == "" {
		return nil
	}

	conf := computeConfidence(tenant.RequestCount, changePct, (cfg.Multiplier-1)*100, false)

	return &Regression{
		Algorithm:     AlgoTenantOutlier,
		Severity:      sev,
		Confidence:    conf,
		Service:       fleet.Service,
		Action:        fleet.Action,
		Title:         fmt.Sprintf("Tenant %s P99 is %.0fms vs fleet %.0fms (%s)", tenant.Service, tenant.P99Ms, fleet.P99Ms, fleet.Service),
		BeforeValue:   fleet.P99Ms,
		AfterValue:    tenant.P99Ms,
		ChangePercent: changePct,
		SampleSize:    tenant.RequestCount,
		Status:        "active",
	}
}

// detectCacheMiss checks if the cache miss rate increased beyond the configured threshold in percentage points.
func detectCacheMiss(before, after *WindowMetrics, cfg CacheMissConfig) *Regression {
	sampleSize := minInt64(before.RequestCount, after.RequestCount)
	if sampleSize < int64(cfg.MinSampleSize) {
		return nil
	}

	diffPP := (after.CacheMissRate - before.CacheMissRate) * 100
	if diffPP <= cfg.ThresholdPP {
		return nil
	}

	baseRate := math.Max(before.CacheMissRate, 0.001)
	changePct := (after.CacheMissRate - before.CacheMissRate) / baseRate * 100

	sev := classifySeverity(changePct)
	if sev == "" {
		return nil
	}

	conf := computeConfidence(sampleSize, changePct, cfg.ThresholdPP, false)

	return &Regression{
		Algorithm:     AlgoCacheMiss,
		Severity:      sev,
		Confidence:    conf,
		Service:       before.Service,
		Action:        before.Action,
		Title:         fmt.Sprintf("Cache miss rate increased %.1f pp (%s)", diffPP, before.Service),
		BeforeValue:   before.CacheMissRate,
		AfterValue:    after.CacheMissRate,
		ChangePercent: changePct,
		SampleSize:    sampleSize,
		Status:        "active",
	}
}

// detectDBFanout checks if the average span count (DB query fanout) increased beyond the configured threshold.
func detectDBFanout(before, after *WindowMetrics, cfg FanoutConfig) *Regression {
	sampleSize := minInt64(before.RequestCount, after.RequestCount)
	if sampleSize < int64(cfg.MinSampleSize) {
		return nil
	}

	if before.AvgSpanCount == 0 {
		return nil
	}

	changePct := (after.AvgSpanCount - before.AvgSpanCount) / before.AvgSpanCount * 100
	if changePct <= cfg.ThresholdPercent {
		return nil
	}

	sev := classifySeverity(changePct)
	if sev == "" {
		return nil
	}

	conf := computeConfidence(sampleSize, changePct, cfg.ThresholdPercent, false)

	return &Regression{
		Algorithm:     AlgoDBFanout,
		Severity:      sev,
		Confidence:    conf,
		Service:       before.Service,
		Action:        before.Action,
		Title:         fmt.Sprintf("DB fanout increased %.0f%% (%s)", changePct, before.Service),
		BeforeValue:   before.AvgSpanCount,
		AfterValue:    after.AvgSpanCount,
		ChangePercent: changePct,
		SampleSize:    sampleSize,
		Status:        "active",
	}
}

// detectPayloadGrowth checks if the average response size increased beyond the configured threshold.
func detectPayloadGrowth(before, after *WindowMetrics, cfg PayloadConfig) *Regression {
	sampleSize := minInt64(before.RequestCount, after.RequestCount)
	if sampleSize < int64(cfg.MinSampleSize) {
		return nil
	}

	if before.AvgRespSize == 0 {
		return nil
	}

	changePct := (after.AvgRespSize - before.AvgRespSize) / before.AvgRespSize * 100
	if changePct <= cfg.ThresholdPercent {
		return nil
	}

	sev := classifySeverity(changePct)
	if sev == "" {
		return nil
	}

	conf := computeConfidence(sampleSize, changePct, cfg.ThresholdPercent, false)

	return &Regression{
		Algorithm:     AlgoPayloadGrowth,
		Severity:      sev,
		Confidence:    conf,
		Service:       before.Service,
		Action:        before.Action,
		Title:         fmt.Sprintf("Response payload grew %.0f%% (%s)", changePct, before.Service),
		BeforeValue:   before.AvgRespSize,
		AfterValue:    after.AvgRespSize,
		ChangePercent: changePct,
		SampleSize:    sampleSize,
		Status:        "active",
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
