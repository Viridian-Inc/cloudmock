package anomaly

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"
)

// Baseline represents learned normal behaviour for a service metric.
type Baseline struct {
	Service     string    `json:"service"`
	Metric      string    `json:"metric"`      // "latency_p50", "error_rate", "throughput"
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"std_dev"`
	SampleCount int       `json:"sample_count"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
}

// Anomaly represents a detected deviation from baseline behaviour.
type Anomaly struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	Metric      string    `json:"metric"`
	Observed    float64   `json:"observed"`
	Expected    float64   `json:"expected"`    // baseline mean
	Deviation   float64   `json:"deviation"`   // how many std devs away
	Severity    string    `json:"severity"`    // "info" (<2sigma), "warning" (2-3sigma), "critical" (>3sigma)
	DetectedAt  time.Time `json:"detected_at"`
	Description string    `json:"description"`
}

// Detector maintains rolling baselines and detects anomalies.
type Detector struct {
	mu        sync.RWMutex
	baselines map[string]*Baseline // key: "service:metric"
	anomalies []Anomaly
	window    time.Duration
	threshold float64 // std devs for alert (default 2.0)
	maxAnoms  int     // max anomalies to keep
}

// NewDetector creates an anomaly detector with the given window and threshold.
func NewDetector(window time.Duration, threshold float64) *Detector {
	if window <= 0 {
		window = 7 * 24 * time.Hour // default 7 days
	}
	if threshold <= 0 {
		threshold = 2.0
	}
	return &Detector{
		baselines: make(map[string]*Baseline),
		window:    window,
		threshold: threshold,
		maxAnoms:  1000,
	}
}

// baselineKey returns the map key for a service+metric pair.
func baselineKey(service, metric string) string {
	return service + ":" + metric
}

// UpdateBaseline performs a rolling update of baseline statistics using
// Welford's online algorithm for numerically stable mean and variance.
func (d *Detector) UpdateBaseline(service, metric string, value float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := baselineKey(service, metric)
	b, ok := d.baselines[key]
	if !ok {
		b = &Baseline{
			Service:     service,
			Metric:      metric,
			WindowStart: time.Now(),
		}
		d.baselines[key] = b
	}

	now := time.Now()

	// If the window has expired, start a new baseline but keep a decayed
	// version of the old stats to avoid losing all context.
	if b.SampleCount > 0 && now.Sub(b.WindowStart) > d.window {
		// Decay: keep 50% of the old statistics.
		b.SampleCount = b.SampleCount / 2
		if b.SampleCount < 1 {
			b.SampleCount = 1
		}
		b.WindowStart = now.Add(-d.window / 2)
	}

	b.SampleCount++
	b.WindowEnd = now

	// Welford's online algorithm.
	delta := value - b.Mean
	b.Mean += delta / float64(b.SampleCount)
	delta2 := value - b.Mean
	// Maintain running sum of squared differences in StdDev field temporarily.
	// We store the variance * (n-1) and compute stddev on read.
	// But for simplicity with the API contract, we compute it directly:
	if b.SampleCount < 2 {
		b.StdDev = 0
	} else {
		// Update variance using online formula.
		// We need to track M2; let's derive it from current stddev.
		// M2_old = stdDev_old^2 * (n-1-1) = stdDev_old^2 * (n-2)
		// Actually, it's cleaner to just store M2 separately. Let's use
		// a simpler approach: keep running sum of squares.
		oldVariance := b.StdDev * b.StdDev
		oldM2 := oldVariance * float64(b.SampleCount-1)
		newM2 := oldM2 + delta*delta2
		b.StdDev = math.Sqrt(newM2 / float64(b.SampleCount))
	}
}

// Check evaluates whether the given value is anomalous for the service+metric.
// Returns nil if the value is within normal bounds or if there's insufficient data.
func (d *Detector) Check(service, metric string, value float64) *Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := baselineKey(service, metric)
	b, ok := d.baselines[key]
	if !ok || b.SampleCount < 10 {
		return nil // insufficient data
	}

	if b.StdDev == 0 {
		return nil // no variance observed
	}

	deviation := math.Abs(value-b.Mean) / b.StdDev
	if deviation < d.threshold {
		return nil
	}

	severity := classifySeverity(deviation)

	anom := Anomaly{
		ID:         generateID(),
		Service:    service,
		Metric:     metric,
		Observed:   value,
		Expected:   b.Mean,
		Deviation:  math.Round(deviation*100) / 100,
		Severity:   severity,
		DetectedAt: time.Now(),
		Description: fmt.Sprintf("%s.%s: observed %.2f, expected %.2f (%.1f sigma)",
			service, metric, value, b.Mean, deviation),
	}

	d.anomalies = append(d.anomalies, anom)
	if len(d.anomalies) > d.maxAnoms {
		d.anomalies = d.anomalies[len(d.anomalies)-d.maxAnoms:]
	}

	return &anom
}

// GetBaselines returns a copy of all current baselines.
func (d *Detector) GetBaselines() []Baseline {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]Baseline, 0, len(d.baselines))
	for _, b := range d.baselines {
		result = append(result, *b)
	}
	return result
}

// GetAnomalies returns anomalies detected within the last N minutes.
func (d *Detector) GetAnomalies(minutes int) []Anomaly {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var result []Anomaly
	for i := len(d.anomalies) - 1; i >= 0; i-- {
		a := d.anomalies[i]
		if a.DetectedAt.Before(cutoff) {
			break
		}
		result = append(result, a)
	}
	return result
}

// WhatChanged returns a human-readable explanation of what might have caused
// an anomaly, based on recent baseline changes across related metrics.
func (d *Detector) WhatChanged(anom Anomaly) string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var changes []string

	// Look for correlated anomalies in the same service within the last 5 minutes.
	cutoff := anom.DetectedAt.Add(-5 * time.Minute)
	for i := len(d.anomalies) - 1; i >= 0; i-- {
		a := d.anomalies[i]
		if a.DetectedAt.Before(cutoff) {
			break
		}
		if a.Service == anom.Service && a.Metric != anom.Metric && a.ID != anom.ID {
			changes = append(changes, fmt.Sprintf("correlated anomaly in %s (%.1f sigma)", a.Metric, a.Deviation))
		}
	}

	// Check for related metrics that have shifted baselines.
	for key, b := range d.baselines {
		if b.Service != anom.Service {
			continue
		}
		if b.Metric == anom.Metric {
			continue
		}
		// If another metric's baseline has high variance, mention it.
		if b.StdDev > 0 && b.Mean > 0 {
			cv := b.StdDev / b.Mean // coefficient of variation
			if cv > 0.5 {
				changes = append(changes, fmt.Sprintf("high variance in %s (cv=%.2f)", key, cv))
			}
		}
	}

	if len(changes) == 0 {
		return fmt.Sprintf("no correlated changes found for %s.%s anomaly", anom.Service, anom.Metric)
	}

	result := fmt.Sprintf("possible causes for %s.%s anomaly: ", anom.Service, anom.Metric)
	for i, c := range changes {
		if i > 0 {
			result += "; "
		}
		result += c
	}
	return result
}

// classifySeverity maps a deviation (in standard deviations) to a severity level.
func classifySeverity(deviation float64) string {
	switch {
	case deviation >= 3.0:
		return "critical"
	case deviation >= 2.0:
		return "warning"
	default:
		return "info"
	}
}

// generateID creates a short random ID.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "anom_" + hex.EncodeToString(b)
}
