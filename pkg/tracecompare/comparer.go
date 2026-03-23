package tracecompare

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// Comparer compares two traces or a trace against a synthesised baseline.
type Comparer struct {
	traces dataplane.TraceReader
}

// New returns a Comparer backed by the given TraceReader.
func New(traces dataplane.TraceReader) *Comparer {
	return &Comparer{traces: traces}
}

// flatSpan is an internal representation of a single span used for alignment.
type flatSpan struct {
	Service    string
	Action     string
	DurationMs float64
	StatusCode int
	Error      string
	Metadata   map[string]string
}

// flattenTrace recursively walks the trace tree in pre-order, collecting every
// span into a flat slice sorted by the traversal order (which mirrors start
// time for well-formed traces).
func flattenTrace(tc *dataplane.TraceContext) []flatSpan {
	if tc == nil {
		return nil
	}
	var out []flatSpan
	out = append(out, flatSpan{
		Service:    tc.Service,
		Action:     tc.Action,
		DurationMs: tc.DurationMs,
		StatusCode: tc.StatusCode,
		Error:      tc.Error,
		Metadata:   tc.Metadata,
	})
	for _, child := range tc.Children {
		out = append(out, flattenTrace(child)...)
	}
	return out
}

// spanKey returns the alignment key for a span.
func spanKey(s flatSpan) string {
	return s.Service + ":" + s.Action
}

// alignSpans matches spans from two flat lists by (service, action). Spans
// with duplicate keys are consumed in order.
func alignSpans(a, b []flatSpan) (matches []SpanMatch, onlyA, onlyB []SpanSummary) {
	indexA := buildSpanIndex(a)
	indexB := buildSpanIndex(b)

	// Track which keys we have already processed.
	seen := make(map[string]bool)

	// Walk A-side keys in order of first appearance.
	for _, sp := range a {
		key := spanKey(sp)
		if seen[key] {
			continue
		}
		seen[key] = true

		listA := indexA[key]
		listB := indexB[key]

		minLen := len(listA)
		if len(listB) < minLen {
			minLen = len(listB)
		}

		for i := 0; i < minLen; i++ {
			matches = append(matches, buildMatch(listA[i], listB[i]))
		}
		for i := minLen; i < len(listA); i++ {
			onlyA = append(onlyA, toSummary(listA[i]))
		}
		for i := minLen; i < len(listB); i++ {
			onlyB = append(onlyB, toSummary(listB[i]))
		}
	}

	// Pick up keys that exist only in B.
	for _, sp := range b {
		key := spanKey(sp)
		if seen[key] {
			continue
		}
		seen[key] = true
		for _, s := range indexB[key] {
			onlyB = append(onlyB, toSummary(s))
		}
	}

	return matches, onlyA, onlyB
}

func buildSpanIndex(spans []flatSpan) map[string][]flatSpan {
	idx := make(map[string][]flatSpan)
	for _, s := range spans {
		key := spanKey(s)
		idx[key] = append(idx[key], s)
	}
	return idx
}

func buildMatch(a, b flatSpan) SpanMatch {
	delta := b.DurationMs - a.DurationMs
	var pct float64
	if a.DurationMs != 0 {
		pct = (delta / a.DurationMs) * 100
	}
	m := SpanMatch{
		Service:      a.Service,
		Action:       a.Action,
		A:            toStats(a),
		B:            toStats(b),
		LatencyDelta: delta,
		LatencyPct:   pct,
		StatusChange: a.StatusCode != b.StatusCode,
		MetadataDiff: diffMetadata(a.Metadata, b.Metadata),
	}
	return m
}

func diffMetadata(a, b map[string]string) map[string][2]string {
	diff := make(map[string][2]string)
	for k, va := range a {
		if vb, ok := b[k]; ok {
			if va != vb {
				diff[k] = [2]string{va, vb}
			}
		} else {
			diff[k] = [2]string{va, ""}
		}
	}
	for k, vb := range b {
		if _, ok := a[k]; !ok {
			diff[k] = [2]string{"", vb}
		}
	}
	if len(diff) == 0 {
		return nil
	}
	return diff
}

func toStats(s flatSpan) SpanStats {
	return SpanStats{
		DurationMs: s.DurationMs,
		StatusCode: s.StatusCode,
		Error:      s.Error,
		Metadata:   s.Metadata,
	}
}

func toSummary(s flatSpan) SpanSummary {
	return SpanSummary{
		Service:    s.Service,
		Action:     s.Action,
		DurationMs: s.DurationMs,
		StatusCode: s.StatusCode,
	}
}

// Compare fetches two traces by ID and produces a span-level comparison.
func (c *Comparer) Compare(ctx context.Context, traceAID, traceBID string) (*TraceComparison, error) {
	traceA, err := c.traces.Get(ctx, traceAID)
	if err != nil {
		return nil, fmt.Errorf("fetching trace A (%s): %w", traceAID, err)
	}
	traceB, err := c.traces.Get(ctx, traceBID)
	if err != nil {
		return nil, fmt.Errorf("fetching trace B (%s): %w", traceBID, err)
	}

	flatA := flattenTrace(traceA)
	flatB := flattenTrace(traceB)
	matches, onlyA, onlyB := alignSpans(flatA, flatB)

	summary := buildSummary(traceA, traceB, matches, onlyA, onlyB)

	return &TraceComparison{
		TraceA:  traceAID,
		TraceB:  traceBID,
		Matches: matches,
		OnlyInA: onlyA,
		OnlyInB: onlyB,
		Summary: summary,
	}, nil
}

func buildSummary(a, b *dataplane.TraceContext, matches []SpanMatch, onlyA, onlyB []SpanSummary) ComparisonSummary {
	var totalA, totalB float64
	if a != nil {
		totalA = a.DurationMs
	}
	if b != nil {
		totalB = b.DurationMs
	}

	var slower, faster int
	var critService string
	var critDelta float64

	for _, m := range matches {
		if m.LatencyDelta > 0 {
			slower++
		} else if m.LatencyDelta < 0 {
			faster++
		}
		if math.Abs(m.LatencyDelta) > critDelta && m.LatencyDelta > 0 {
			critDelta = math.Abs(m.LatencyDelta)
			critService = m.Service + ":" + m.Action
		}
	}

	return ComparisonSummary{
		TotalLatencyA: totalA,
		TotalLatencyB: totalB,
		LatencyDelta:  totalB - totalA,
		SlowerSpans:   slower,
		FasterSpans:   faster,
		AddedSpans:    len(onlyB),
		RemovedSpans:  len(onlyA),
		CriticalPath:  critService,
	}
}

// CompareBaseline compares a trace against a synthesised P50 baseline derived
// from recent similar traces (same root service).
func (c *Comparer) CompareBaseline(ctx context.Context, traceID string) (*TraceComparison, error) {
	target, err := c.traces.Get(ctx, traceID)
	if err != nil {
		return nil, fmt.Errorf("fetching target trace (%s): %w", traceID, err)
	}

	results, err := c.traces.Search(ctx, dataplane.TraceFilter{
		Service: target.Service,
		Limit:   20,
	})
	if err != nil {
		return nil, fmt.Errorf("searching for similar traces: %w", err)
	}

	// Exclude the target trace itself.
	var similar []dataplane.TraceSummary
	for _, r := range results {
		if r.TraceID != traceID {
			similar = append(similar, r)
		}
	}
	if len(similar) == 0 {
		return nil, errors.New("no similar traces found for baseline")
	}

	// Collect spans from all similar traces grouped by (service, action).
	type spanGroup struct {
		durations  []float64
		statusCode map[int]int // code -> count
		errors     map[string]int
	}
	groups := make(map[string]*spanGroup)

	for _, ts := range similar {
		tc, err := c.traces.Get(ctx, ts.TraceID)
		if err != nil {
			continue // skip traces we can't fetch
		}
		for _, fs := range flattenTrace(tc) {
			key := spanKey(fs)
			g, ok := groups[key]
			if !ok {
				g = &spanGroup{
					statusCode: make(map[int]int),
					errors:     make(map[string]int),
				}
				groups[key] = g
			}
			g.durations = append(g.durations, fs.DurationMs)
			g.statusCode[fs.StatusCode]++
			if fs.Error != "" {
				g.errors[fs.Error]++
			}
		}
	}

	// Build synthetic baseline spans from P50 values.
	var baseline []flatSpan
	for key, g := range groups {
		svc, action := splitKey(key)
		sort.Float64s(g.durations)
		p50 := median(g.durations)
		code := mostCommon(g.statusCode)
		baseline = append(baseline, flatSpan{
			Service:    svc,
			Action:     action,
			DurationMs: p50,
			StatusCode: code,
		})
	}

	// Sort baseline for deterministic output.
	sort.Slice(baseline, func(i, j int) bool {
		if baseline[i].Service != baseline[j].Service {
			return baseline[i].Service < baseline[j].Service
		}
		return baseline[i].Action < baseline[j].Action
	})

	flatTarget := flattenTrace(target)
	matches, onlyA, onlyB := alignSpans(baseline, flatTarget)

	// Compute summary: baseline is "A", target is "B".
	var baselineTotal float64
	for _, s := range baseline {
		baselineTotal += s.DurationMs
	}
	summary := ComparisonSummary{
		TotalLatencyA: baselineTotal,
		TotalLatencyB: target.DurationMs,
		LatencyDelta:  target.DurationMs - baselineTotal,
		AddedSpans:    len(onlyB),
		RemovedSpans:  len(onlyA),
	}

	var critService string
	var critDelta float64
	for _, m := range matches {
		if m.LatencyDelta > 0 {
			summary.SlowerSpans++
		} else if m.LatencyDelta < 0 {
			summary.FasterSpans++
		}
		if math.Abs(m.LatencyDelta) > critDelta && m.LatencyDelta > 0 {
			critDelta = math.Abs(m.LatencyDelta)
			critService = m.Service + ":" + m.Action
		}
	}
	summary.CriticalPath = critService

	return &TraceComparison{
		TraceA:  "baseline",
		TraceB:  traceID,
		Matches: matches,
		OnlyInA: onlyA,
		OnlyInB: onlyB,
		Summary: summary,
	}, nil
}

func splitKey(key string) (string, string) {
	for i, c := range key {
		if c == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func mostCommon(counts map[int]int) int {
	var best, bestCount int
	for k, v := range counts {
		if v > bestCount {
			best = k
			bestCount = v
		}
	}
	return best
}
