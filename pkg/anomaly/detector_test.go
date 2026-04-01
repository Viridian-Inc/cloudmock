package anomaly

import (
	"math"
	"testing"
	"time"
)

func TestBaselineCalculation(t *testing.T) {
	d := NewDetector(24*time.Hour, 2.0)

	// Feed known values: 10, 20, 30, 40, 50
	// Mean should be 30, StdDev ~= 14.14 (population)
	values := []float64{10, 20, 30, 40, 50}
	for _, v := range values {
		d.UpdateBaseline("svc-a", "latency_p50", v)
	}

	baselines := d.GetBaselines()
	if len(baselines) != 1 {
		t.Fatalf("expected 1 baseline, got %d", len(baselines))
	}

	b := baselines[0]
	if b.Service != "svc-a" {
		t.Errorf("expected service svc-a, got %s", b.Service)
	}
	if b.Metric != "latency_p50" {
		t.Errorf("expected metric latency_p50, got %s", b.Metric)
	}
	if b.SampleCount != 5 {
		t.Errorf("expected 5 samples, got %d", b.SampleCount)
	}
	if math.Abs(b.Mean-30.0) > 0.01 {
		t.Errorf("expected mean 30.0, got %.2f", b.Mean)
	}
	// Population stddev of [10,20,30,40,50] = sqrt(200) ~= 14.14
	expectedStdDev := math.Sqrt(200.0)
	if math.Abs(b.StdDev-expectedStdDev) > 0.5 {
		t.Errorf("expected stddev ~%.2f, got %.2f", expectedStdDev, b.StdDev)
	}
}

func TestAnomalyDetectionAboveThreshold(t *testing.T) {
	d := NewDetector(24*time.Hour, 2.0)

	// Build a baseline with 20 samples around mean=100, stddev~=5.
	for i := 0; i < 20; i++ {
		d.UpdateBaseline("api", "latency_p50", 100+float64(i%5)-2)
	}

	// Value far above the mean should be anomalous.
	anom := d.Check("api", "latency_p50", 200.0)
	if anom == nil {
		t.Fatal("expected anomaly, got nil")
	}
	if anom.Service != "api" {
		t.Errorf("expected service api, got %s", anom.Service)
	}
	if anom.Observed != 200.0 {
		t.Errorf("expected observed 200.0, got %.2f", anom.Observed)
	}
	if anom.Deviation < 2.0 {
		t.Errorf("expected deviation >= 2.0, got %.2f", anom.Deviation)
	}
}

func TestAnomalyDetectionBelowThreshold(t *testing.T) {
	d := NewDetector(24*time.Hour, 2.0)

	// Build a baseline.
	for i := 0; i < 20; i++ {
		d.UpdateBaseline("api", "latency_p50", 100+float64(i%5)-2)
	}

	// Value within normal range should not be anomalous.
	anom := d.Check("api", "latency_p50", 101.0)
	if anom != nil {
		t.Errorf("expected no anomaly, got %+v", anom)
	}
}

func TestSeverityClassification(t *testing.T) {
	tests := []struct {
		deviation float64
		expected  string
	}{
		{1.5, "info"},
		{2.0, "warning"},
		{2.5, "warning"},
		{3.0, "critical"},
		{5.0, "critical"},
	}

	for _, tt := range tests {
		got := classifySeverity(tt.deviation)
		if got != tt.expected {
			t.Errorf("classifySeverity(%.1f) = %s, want %s", tt.deviation, got, tt.expected)
		}
	}
}

func TestInsufficientData(t *testing.T) {
	d := NewDetector(24*time.Hour, 2.0)

	// Only 5 samples, detector requires 10.
	for i := 0; i < 5; i++ {
		d.UpdateBaseline("svc", "metric", float64(i))
	}

	anom := d.Check("svc", "metric", 1000.0)
	if anom != nil {
		t.Error("expected nil anomaly with insufficient data")
	}
}

func TestGetAnomalies(t *testing.T) {
	d := NewDetector(24*time.Hour, 2.0)

	// Build baseline.
	for i := 0; i < 20; i++ {
		d.UpdateBaseline("svc", "latency", 100.0)
	}
	// Force some variance.
	d.UpdateBaseline("svc", "latency", 105.0)
	d.UpdateBaseline("svc", "latency", 95.0)

	// Trigger anomaly.
	d.Check("svc", "latency", 500.0)

	anoms := d.GetAnomalies(60)
	if len(anoms) != 1 {
		t.Fatalf("expected 1 anomaly, got %d", len(anoms))
	}
	if anoms[0].Service != "svc" {
		t.Errorf("expected service svc, got %s", anoms[0].Service)
	}
}

func TestRollingWindowDecay(t *testing.T) {
	// Use a tiny window so it expires immediately.
	d := NewDetector(1*time.Millisecond, 2.0)

	for i := 0; i < 20; i++ {
		d.UpdateBaseline("svc", "metric", 100.0)
	}

	// Wait for window to expire.
	time.Sleep(5 * time.Millisecond)

	// Update should trigger decay.
	d.UpdateBaseline("svc", "metric", 100.0)

	baselines := d.GetBaselines()
	if len(baselines) != 1 {
		t.Fatalf("expected 1 baseline, got %d", len(baselines))
	}
	// Sample count should have been decayed (halved from 20 to 10, then +1 = 11).
	if baselines[0].SampleCount > 15 {
		t.Errorf("expected decayed sample count, got %d", baselines[0].SampleCount)
	}
}

func TestWhatChanged(t *testing.T) {
	d := NewDetector(24*time.Hour, 2.0)

	// Build baselines for two metrics.
	for i := 0; i < 20; i++ {
		d.UpdateBaseline("api", "latency", 100.0)
		d.UpdateBaseline("api", "error_rate", 0.01)
	}
	// Add some variance.
	d.UpdateBaseline("api", "latency", 105.0)
	d.UpdateBaseline("api", "latency", 95.0)
	d.UpdateBaseline("api", "error_rate", 0.02)
	d.UpdateBaseline("api", "error_rate", 0.005)

	// Trigger anomalies in both metrics.
	d.Check("api", "error_rate", 5.0)
	anom := d.Check("api", "latency", 500.0)
	if anom == nil {
		t.Fatal("expected anomaly")
	}

	explanation := d.WhatChanged(*anom)
	if explanation == "" {
		t.Error("expected non-empty explanation")
	}
}

func TestMultipleServicesBaselines(t *testing.T) {
	d := NewDetector(24*time.Hour, 2.0)

	d.UpdateBaseline("svc-a", "latency", 100.0)
	d.UpdateBaseline("svc-b", "latency", 200.0)

	baselines := d.GetBaselines()
	if len(baselines) != 2 {
		t.Errorf("expected 2 baselines, got %d", len(baselines))
	}
}
