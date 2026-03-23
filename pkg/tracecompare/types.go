package tracecompare

// TraceComparison is the result of comparing two traces span-by-span.
type TraceComparison struct {
	TraceA  string            `json:"trace_a"`
	TraceB  string            `json:"trace_b"`
	Matches []SpanMatch       `json:"matches"`
	OnlyInA []SpanSummary     `json:"only_in_a"`
	OnlyInB []SpanSummary     `json:"only_in_b"`
	Summary ComparisonSummary `json:"summary"`
}

// SpanMatch represents a pair of spans from two traces that share the same
// (service, action) key, along with computed deltas.
type SpanMatch struct {
	Service      string               `json:"service"`
	Action       string               `json:"action"`
	A            SpanStats            `json:"a"`
	B            SpanStats            `json:"b"`
	LatencyDelta float64              `json:"latency_delta_ms"`
	LatencyPct   float64              `json:"latency_pct"`
	StatusChange bool                 `json:"status_change"`
	MetadataDiff map[string][2]string `json:"metadata_diff,omitempty"`
}

// SpanStats captures the key measurements of a single span.
type SpanStats struct {
	DurationMs float64           `json:"duration_ms"`
	StatusCode int               `json:"status_code"`
	Error      string            `json:"error,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// SpanSummary is a lightweight description of a span that exists in only one
// side of a comparison.
type SpanSummary struct {
	Service    string  `json:"service"`
	Action     string  `json:"action"`
	DurationMs float64 `json:"duration_ms"`
	StatusCode int     `json:"status_code"`
}

// ComparisonSummary aggregates high-level statistics for a trace comparison.
type ComparisonSummary struct {
	TotalLatencyA float64 `json:"total_latency_a_ms"`
	TotalLatencyB float64 `json:"total_latency_b_ms"`
	LatencyDelta  float64 `json:"latency_delta_ms"`
	SlowerSpans   int     `json:"slower_spans"`
	FasterSpans   int     `json:"faster_spans"`
	AddedSpans    int     `json:"added_spans"`
	RemovedSpans  int     `json:"removed_spans"`
	CriticalPath  string  `json:"critical_path,omitempty"`
}
