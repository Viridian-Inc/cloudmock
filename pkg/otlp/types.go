package otlp

import "fmt"

// OTLP JSON types for traces, metrics, and logs ingestion.
// These mirror the OpenTelemetry proto definitions but use JSON-friendly
// Go types so we can decode OTLP/HTTP JSON payloads without a protobuf dependency.

// ── Traces ──────────────────────────────────────────────────────────────────

// ExportTraceRequest is the top-level OTLP trace export payload.
type ExportTraceRequest struct {
	ResourceSpans []ResourceSpan `json:"resourceSpans"`
}

// ResourceSpan groups spans by originating resource.
type ResourceSpan struct {
	Resource   Resource    `json:"resource"`
	ScopeSpans []ScopeSpan `json:"scopeSpans"`
}

// ScopeSpan groups spans by instrumentation scope.
type ScopeSpan struct {
	Scope InstrumentationScope `json:"scope"`
	Spans []Span               `json:"spans"`
}

// Span represents a single OTLP span.
type Span struct {
	TraceID            string      `json:"traceId"`
	SpanID             string      `json:"spanId"`
	ParentSpanID       string      `json:"parentSpanId"`
	Name               string      `json:"name"`
	Kind               int         `json:"kind"` // 0=unspecified,1=internal,2=server,3=client,4=producer,5=consumer
	StartTimeUnixNano  string      `json:"startTimeUnixNano"`
	EndTimeUnixNano    string      `json:"endTimeUnixNano"`
	Attributes         []KeyValue  `json:"attributes"`
	Status             SpanStatus  `json:"status"`
	Events             []SpanEvent `json:"events"`
}

// SpanStatus represents the status of a span.
type SpanStatus struct {
	Code    int    `json:"code"` // 0=unset,1=ok,2=error
	Message string `json:"message"`
}

// SpanEvent represents a timed event within a span.
type SpanEvent struct {
	TimeUnixNano string     `json:"timeUnixNano"`
	Name         string     `json:"name"`
	Attributes   []KeyValue `json:"attributes"`
}

// ── Metrics ─────────────────────────────────────────────────────────────────

// ExportMetricsRequest is the top-level OTLP metrics export payload.
type ExportMetricsRequest struct {
	ResourceMetrics []ResourceMetric `json:"resourceMetrics"`
}

// ResourceMetric groups metrics by originating resource.
type ResourceMetric struct {
	Resource     Resource      `json:"resource"`
	ScopeMetrics []ScopeMetric `json:"scopeMetrics"`
}

// ScopeMetric groups metrics by instrumentation scope.
type ScopeMetric struct {
	Scope   InstrumentationScope `json:"scope"`
	Metrics []Metric             `json:"metrics"`
}

// Metric represents a single OTLP metric.
type Metric struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Unit        string     `json:"unit"`
	Gauge       *Gauge     `json:"gauge,omitempty"`
	Sum         *Sum       `json:"sum,omitempty"`
	Histogram   *Histogram `json:"histogram,omitempty"`
}

// Gauge represents a gauge metric.
type Gauge struct {
	DataPoints []NumberDataPoint `json:"dataPoints"`
}

// Sum represents a sum (counter) metric.
type Sum struct {
	DataPoints             []NumberDataPoint `json:"dataPoints"`
	AggregationTemporality int               `json:"aggregationTemporality"`
	IsMonotonic            bool              `json:"isMonotonic"`
}

// Histogram represents a histogram metric.
type Histogram struct {
	DataPoints             []HistogramDataPoint `json:"dataPoints"`
	AggregationTemporality int                  `json:"aggregationTemporality"`
}

// NumberDataPoint is a single data point for gauge/sum metrics.
type NumberDataPoint struct {
	Attributes   []KeyValue `json:"attributes"`
	TimeUnixNano string     `json:"timeUnixNano"`
	AsDouble     *float64   `json:"asDouble,omitempty"`
	AsInt        *string    `json:"asInt,omitempty"` // OTLP JSON encodes int64 as string
}

// HistogramDataPoint is a single data point for histogram metrics.
type HistogramDataPoint struct {
	Attributes     []KeyValue `json:"attributes"`
	TimeUnixNano   string     `json:"timeUnixNano"`
	Count          uint64     `json:"count"`
	Sum            *float64   `json:"sum,omitempty"`
	BucketCounts   []uint64   `json:"bucketCounts"`
	ExplicitBounds []float64  `json:"explicitBounds"`
	Min            *float64   `json:"min,omitempty"`
	Max            *float64   `json:"max,omitempty"`
}

// ── Logs ────────────────────────────────────────────────────────────────────

// ExportLogsRequest is the top-level OTLP logs export payload.
type ExportLogsRequest struct {
	ResourceLogs []ResourceLog `json:"resourceLogs"`
}

// ResourceLog groups log records by originating resource.
type ResourceLog struct {
	Resource  Resource   `json:"resource"`
	ScopeLogs []ScopeLog `json:"scopeLogs"`
}

// ScopeLog groups log records by instrumentation scope.
type ScopeLog struct {
	Scope      InstrumentationScope `json:"scope"`
	LogRecords []LogRecord          `json:"logRecords"`
}

// LogRecord represents a single OTLP log record.
type LogRecord struct {
	TimeUnixNano         string     `json:"timeUnixNano"`
	ObservedTimeUnixNano string     `json:"observedTimeUnixNano"`
	SeverityNumber       int        `json:"severityNumber"`
	SeverityText         string     `json:"severityText"`
	Body                 AnyValue   `json:"body"`
	Attributes           []KeyValue `json:"attributes"`
	TraceID              string     `json:"traceId"`
	SpanID               string     `json:"spanId"`
}

// ── Common ──────────────────────────────────────────────────────────────────

// Resource describes the entity producing telemetry.
type Resource struct {
	Attributes []KeyValue `json:"attributes"`
}

// InstrumentationScope identifies the instrumentation library.
type InstrumentationScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// KeyValue is an OTLP attribute key-value pair.
type KeyValue struct {
	Key   string   `json:"key"`
	Value AnyValue `json:"value"`
}

// AnyValue is an OTLP polymorphic value.
type AnyValue struct {
	StringValue *string  `json:"stringValue,omitempty"`
	IntValue    *string  `json:"intValue,omitempty"`    // JSON encodes int64 as string
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
}

// StringVal returns the string representation of the value.
func (v AnyValue) StringVal() string {
	if v.StringValue != nil {
		return *v.StringValue
	}
	if v.IntValue != nil {
		return *v.IntValue
	}
	if v.DoubleValue != nil {
		return fmt.Sprintf("%g", *v.DoubleValue)
	}
	if v.BoolValue != nil {
		if *v.BoolValue {
			return "true"
		}
		return "false"
	}
	return ""
}

// AttributeMap converts a slice of KeyValue pairs to a map[string]string.
func AttributeMap(attrs []KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, kv := range attrs {
		m[kv.Key] = kv.Value.StringVal()
	}
	return m
}

// ResourceServiceName extracts the "service.name" attribute from a Resource.
// Returns "unknown" if not found.
func ResourceServiceName(r Resource) string {
	for _, kv := range r.Attributes {
		if kv.Key == "service.name" {
			return kv.Value.StringVal()
		}
	}
	return "unknown"
}
