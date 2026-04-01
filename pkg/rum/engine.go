package rum

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// EngineConfig holds RUM engine configuration.
type EngineConfig struct {
	SampleRate float64 // 0.0–1.0, fraction of sessions to keep
	MaxEvents  int     // circular buffer capacity
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		SampleRate: 1.0,
		MaxEvents:  10000,
	}
}

// Engine processes incoming RUM events: classifies web vitals, fingerprints
// errors, applies session sampling, and writes to the store.
type Engine struct {
	store      RUMStore
	sampleRate float64
	// sampledSessions tracks which sessions have been accepted/rejected.
	// true = accepted, false = rejected.
	sampledSessions map[string]bool
	rng             *rand.Rand
}

// New creates a RUM engine with the given store and configuration.
func New(store RUMStore, cfg EngineConfig) *Engine {
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}
	if cfg.SampleRate > 1.0 {
		cfg.SampleRate = 1.0
	}
	return &Engine{
		store:           store,
		sampleRate:      cfg.SampleRate,
		sampledSessions: make(map[string]bool),
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Store returns the underlying RUM store.
func (e *Engine) Store() RUMStore {
	return e.store
}

// IngestEvent processes and stores a single RUM event.
// Returns false if the event was dropped by session sampling.
func (e *Engine) IngestEvent(event RUMEvent) bool {
	if !e.shouldSample(event.SessionID) {
		return false
	}
	e.enrich(&event)
	_ = e.store.WriteEvent(event)
	return true
}

// IngestBatch processes and stores a batch of RUM events.
// Returns the number of events actually stored (after sampling).
func (e *Engine) IngestBatch(events []RUMEvent) int {
	var accepted []RUMEvent
	for i := range events {
		if !e.shouldSample(events[i].SessionID) {
			continue
		}
		e.enrich(&events[i])
		accepted = append(accepted, events[i])
	}
	if len(accepted) > 0 {
		_ = e.store.WriteBatch(accepted)
	}
	return len(accepted)
}

// shouldSample decides whether a session's events should be kept.
// Once a session is sampled in/out, the decision is sticky.
func (e *Engine) shouldSample(sessionID string) bool {
	if e.sampleRate >= 1.0 {
		return true
	}
	if sessionID == "" {
		return true
	}
	if accepted, ok := e.sampledSessions[sessionID]; ok {
		return accepted
	}
	accepted := e.rng.Float64() < e.sampleRate
	e.sampledSessions[sessionID] = accepted
	return accepted
}

// enrich adds server-side computed fields: web vital ratings and error fingerprints.
func (e *Engine) enrich(event *RUMEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ID == "" {
		event.ID = generateID()
	}

	if event.Type == EventWebVital && event.WebVital != nil {
		event.WebVital.Rating = ClassifyWebVital(event.WebVital.Name, event.WebVital.Value)
	}

	if event.Type == EventJSError && event.JSError != nil {
		event.JSError.Fingerprint = FingerprintError(event.JSError)
	}
}

// --- Web Vitals Classification (Google's thresholds) ---

// ClassifyWebVital returns "good", "needs-improvement", or "poor" for a given
// web vital metric, using Google's Core Web Vitals thresholds.
func ClassifyWebVital(name string, value float64) string {
	switch strings.ToUpper(name) {
	case "LCP":
		// Largest Contentful Paint: good <= 2500ms, poor > 4000ms
		if value <= 2500 {
			return "good"
		}
		if value <= 4000 {
			return "needs-improvement"
		}
		return "poor"
	case "FID":
		// First Input Delay: good <= 100ms, poor > 300ms
		if value <= 100 {
			return "good"
		}
		if value <= 300 {
			return "needs-improvement"
		}
		return "poor"
	case "CLS":
		// Cumulative Layout Shift: good <= 0.1, poor > 0.25
		if value <= 0.1 {
			return "good"
		}
		if value <= 0.25 {
			return "needs-improvement"
		}
		return "poor"
	case "TTFB":
		// Time to First Byte: good <= 800ms, poor > 1800ms
		if value <= 800 {
			return "good"
		}
		if value <= 1800 {
			return "needs-improvement"
		}
		return "poor"
	case "FCP":
		// First Contentful Paint: good <= 1800ms, poor > 3000ms
		if value <= 1800 {
			return "good"
		}
		if value <= 3000 {
			return "needs-improvement"
		}
		return "poor"
	case "INP":
		// Interaction to Next Paint: good <= 200ms, poor > 500ms
		if value <= 200 {
			return "good"
		}
		if value <= 500 {
			return "needs-improvement"
		}
		return "poor"
	default:
		return "good"
	}
}

// FingerprintError computes a stable hash for grouping duplicate JS errors.
// Uses message + source + first non-empty stack frame line.
func FingerprintError(err *JSErrorEvent) string {
	input := err.Message + "|" + err.Source
	if err.Stack != "" {
		// Use first stack frame for grouping
		lines := strings.Split(err.Stack, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && strings.Contains(trimmed, "at ") {
				input += "|" + trimmed
				break
			}
		}
	}
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:8])
}

// generateID creates a short unique ID for events.
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("rum_%x", b)
}
