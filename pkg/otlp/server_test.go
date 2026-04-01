package otlp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/dataplane/memory"
)

func newTestServer() (*Server, *gateway.TraceStore, *gateway.RequestLog) {
	traceStore := gateway.NewTraceStore(500)
	requestLog := gateway.NewRequestLog(1000)
	requestStats := gateway.NewRequestStats()
	bus := eventbus.NewBus()

	dp := &dataplane.DataPlane{
		Traces:   memory.NewTraceStore(traceStore),
		TraceW:   memory.NewTraceStore(traceStore),
		Requests: memory.NewRequestStore(requestLog),
		RequestW: memory.NewRequestStore(requestLog),
		Metrics:  memory.NewMetricStore(requestStats, requestLog),
		MetricW:  memory.NewMetricStore(requestStats, requestLog),
		Mode:     "local",
	}

	srv := NewServer(dp, bus, "us-east-1", "000000000000")
	return srv, traceStore, requestLog
}

func TestHandleTracesBasic(t *testing.T) {
	srv, traceStore, _ := newTestServer()

	payload := `{
		"resourceSpans": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "test-service"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "abc123",
					"spanId": "def456",
					"name": "GET /api/health",
					"startTimeUnixNano": "1711900000000000000",
					"endTimeUnixNano": "1711900001000000000",
					"status": {"code": 1}
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// Verify the span was stored.
	trace := traceStore.Get("abc123")
	if trace == nil {
		t.Fatal("expected trace abc123 to be stored")
	}
	if trace.Service != "test-service" {
		t.Errorf("expected service 'test-service', got %q", trace.Service)
	}
	if trace.Action != "GET /api/health" {
		t.Errorf("expected action 'GET /api/health', got %q", trace.Action)
	}
	if trace.SpanID != "def456" {
		t.Errorf("expected span ID 'def456', got %q", trace.SpanID)
	}
}

func TestHandleTracesWithParentSpan(t *testing.T) {
	srv, traceStore, _ := newTestServer()

	payload := `{
		"resourceSpans": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "my-app"}}]
			},
			"scopeSpans": [{
				"spans": [
					{
						"traceId": "trace-001",
						"spanId": "span-root",
						"name": "HTTP GET /users",
						"startTimeUnixNano": "1711900000000000000",
						"endTimeUnixNano": "1711900002000000000",
						"status": {"code": 1}
					},
					{
						"traceId": "trace-001",
						"spanId": "span-child",
						"parentSpanId": "span-root",
						"name": "DB Query",
						"startTimeUnixNano": "1711900000500000000",
						"endTimeUnixNano": "1711900001500000000",
						"status": {"code": 1}
					}
				]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	trace := traceStore.Get("trace-001")
	if trace == nil {
		t.Fatal("expected trace trace-001 to be stored")
	}
	if trace.SpanID != "span-root" {
		t.Errorf("expected root span 'span-root', got %q", trace.SpanID)
	}
}

func TestHandleTracesErrorSpan(t *testing.T) {
	srv, traceStore, _ := newTestServer()

	payload := `{
		"resourceSpans": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "error-service"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "err-trace",
					"spanId": "err-span",
					"name": "POST /api/fail",
					"startTimeUnixNano": "1711900000000000000",
					"endTimeUnixNano": "1711900000100000000",
					"status": {"code": 2, "message": "connection refused"},
					"attributes": [
						{"key": "http.status_code", "value": {"stringValue": "503"}}
					]
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	trace := traceStore.Get("err-trace")
	if trace == nil {
		t.Fatal("expected trace to be stored")
	}
	if trace.StatusCode != 503 {
		t.Errorf("expected status code 503, got %d", trace.StatusCode)
	}
	if trace.Error != "connection refused" {
		t.Errorf("expected error 'connection refused', got %q", trace.Error)
	}
}

func TestHandleMetrics(t *testing.T) {
	srv, _, _ := newTestServer()

	payload := `{
		"resourceMetrics": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "metrics-svc"}}]
			},
			"scopeMetrics": [{
				"metrics": [{
					"name": "http.server.duration",
					"gauge": {
						"dataPoints": [{
							"timeUnixNano": "1711900000000000000",
							"asDouble": 42.5
						}]
					}
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != "{}" {
		t.Errorf("expected empty JSON object response, got %q", string(body))
	}
}

func TestHandleMetricsSum(t *testing.T) {
	srv, _, _ := newTestServer()

	payload := `{
		"resourceMetrics": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "counter-svc"}}]
			},
			"scopeMetrics": [{
				"metrics": [{
					"name": "http.server.request_count",
					"sum": {
						"dataPoints": [
							{"timeUnixNano": "1711900000000000000", "asInt": "100"},
							{"timeUnixNano": "1711900001000000000", "asInt": "150"}
						],
						"aggregationTemporality": 2,
						"isMonotonic": true
					}
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleLogs(t *testing.T) {
	srv, _, requestLog := newTestServer()

	payload := `{
		"resourceLogs": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "log-service"}}]
			},
			"scopeLogs": [{
				"logRecords": [{
					"timeUnixNano": "1711900000000000000",
					"severityText": "ERROR",
					"severityNumber": 17,
					"body": {"stringValue": "connection timeout after 30s"},
					"traceId": "log-trace-1",
					"spanId": "log-span-1",
					"attributes": [
						{"key": "exception.type", "value": {"stringValue": "TimeoutError"}}
					]
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify the log was stored as a request entry.
	entries := requestLog.Recent("", 10)
	if len(entries) == 0 {
		t.Fatal("expected at least one log entry in request log")
	}

	found := false
	for _, e := range entries {
		if e.TraceID == "log-trace-1" && e.Service == "log-service" {
			found = true
			if e.Action != "log:ERROR" {
				t.Errorf("expected action 'log:ERROR', got %q", e.Action)
			}
			if e.RequestBody != "connection timeout after 30s" {
				t.Errorf("expected body 'connection timeout after 30s', got %q", e.RequestBody)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find log entry with traceId 'log-trace-1'")
	}
}

func TestHandleLogsInfoLevel(t *testing.T) {
	srv, _, requestLog := newTestServer()

	payload := `{
		"resourceLogs": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "info-svc"}}]
			},
			"scopeLogs": [{
				"logRecords": [{
					"timeUnixNano": "1711900000000000000",
					"severityText": "INFO",
					"body": {"stringValue": "server started on port 8080"},
					"traceId": "info-trace"
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	entries := requestLog.Recent("", 10)
	found := false
	for _, e := range entries {
		if e.TraceID == "info-trace" {
			found = true
			if e.StatusCode != 200 {
				t.Errorf("expected status 200 for INFO log, got %d", e.StatusCode)
			}
			if e.Error != "" {
				t.Errorf("expected no error for INFO log, got %q", e.Error)
			}
		}
	}
	if !found {
		t.Error("expected to find info log entry")
	}
}

func TestRejectProtobuf(t *testing.T) {
	srv, _, _ := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader("binary data"))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "protobuf") {
		t.Errorf("expected helpful error about protobuf, got %q", string(body))
	}
	if !strings.Contains(string(body), "http/json") {
		t.Errorf("expected suggestion to use http/json, got %q", string(body))
	}
}

func TestRejectProtobufOnMetrics(t *testing.T) {
	srv, _, _ := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", strings.NewReader("binary data"))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", w.Code)
	}
}

func TestRejectProtobufOnLogs(t *testing.T) {
	srv, _, _ := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader("binary data"))
	req.Header.Set("Content-Type", "application/protobuf")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv, _, _ := newTestServer()

	for _, path := range []string{"/v1/traces", "/v1/metrics", "/v1/logs"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET %s: expected 405, got %d", path, w.Code)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv, _, _ := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
}

func TestInvalidJSON(t *testing.T) {
	srv, _, _ := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestEmptyPayload(t *testing.T) {
	srv, _, _ := newTestServer()

	// Empty but valid JSON.
	payload := `{"resourceSpans":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty payload, got %d", w.Code)
	}
}

func TestMultipleResourceSpans(t *testing.T) {
	srv, traceStore, _ := newTestServer()

	payload := `{
		"resourceSpans": [
			{
				"resource": {
					"attributes": [{"key": "service.name", "value": {"stringValue": "svc-a"}}]
				},
				"scopeSpans": [{"spans": [{
					"traceId": "multi-1", "spanId": "s1", "name": "op-a",
					"startTimeUnixNano": "1711900000000000000", "endTimeUnixNano": "1711900001000000000",
					"status": {"code": 1}
				}]}]
			},
			{
				"resource": {
					"attributes": [{"key": "service.name", "value": {"stringValue": "svc-b"}}]
				},
				"scopeSpans": [{"spans": [{
					"traceId": "multi-2", "spanId": "s2", "name": "op-b",
					"startTimeUnixNano": "1711900000000000000", "endTimeUnixNano": "1711900002000000000",
					"status": {"code": 1}
				}]}]
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if traceStore.Get("multi-1") == nil {
		t.Error("expected trace multi-1")
	}
	if traceStore.Get("multi-2") == nil {
		t.Error("expected trace multi-2")
	}
	if traceStore.Get("multi-1").Service != "svc-a" {
		t.Errorf("expected svc-a, got %q", traceStore.Get("multi-1").Service)
	}
	if traceStore.Get("multi-2").Service != "svc-b" {
		t.Errorf("expected svc-b, got %q", traceStore.Get("multi-2").Service)
	}
}

func TestHTTPAttributeExtraction(t *testing.T) {
	srv, traceStore, _ := newTestServer()

	payload := `{
		"resourceSpans": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "http-svc"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "http-trace",
					"spanId": "http-span",
					"name": "GET /api/users",
					"startTimeUnixNano": "1711900000000000000",
					"endTimeUnixNano": "1711900000500000000",
					"status": {"code": 1},
					"attributes": [
						{"key": "http.method", "value": {"stringValue": "GET"}},
						{"key": "http.target", "value": {"stringValue": "/api/users"}},
						{"key": "http.status_code", "value": {"stringValue": "200"}}
					]
				}]
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	trace := traceStore.Get("http-trace")
	if trace == nil {
		t.Fatal("expected trace to be stored")
	}
	if trace.Method != "GET" {
		t.Errorf("expected method 'GET', got %q", trace.Method)
	}
	if trace.Path != "/api/users" {
		t.Errorf("expected path '/api/users', got %q", trace.Path)
	}
}

// ── Unit tests for helper functions ─────────────────────────────────────────

func TestSeverityToLevel(t *testing.T) {
	tests := []struct {
		text   string
		number int
		want   string
	}{
		{"ERROR", 17, "ERROR"},
		{"WARN", 13, "WARN"},
		{"INFO", 9, "INFO"},
		{"DEBUG", 5, "DEBUG"},
		{"TRACE", 1, "TRACE"},
		{"FATAL", 21, "FATAL"},
		{"", 17, "ERROR"},
		{"", 13, "WARN"},
		{"", 9, "INFO"},
		{"", 5, "DEBUG"},
		{"", 1, "TRACE"},
		{"", 21, "FATAL"},
		{"", 0, "INFO"},
		{"Warning", 0, "WARN"},
	}

	for _, tt := range tests {
		got := severityToLevel(tt.text, tt.number)
		if got != tt.want {
			t.Errorf("severityToLevel(%q, %d) = %q, want %q", tt.text, tt.number, got, tt.want)
		}
	}
}

func TestAttributeMap(t *testing.T) {
	attrs := []KeyValue{
		{Key: "http.method", Value: AnyValue{StringValue: strPtr("GET")}},
		{Key: "http.status_code", Value: AnyValue{IntValue: strPtr("200")}},
		{Key: "latency", Value: AnyValue{DoubleValue: float64Ptr(42.5)}},
		{Key: "cached", Value: AnyValue{BoolValue: boolPtr(true)}},
	}

	m := AttributeMap(attrs)
	if m["http.method"] != "GET" {
		t.Errorf("expected GET, got %q", m["http.method"])
	}
	if m["http.status_code"] != "200" {
		t.Errorf("expected 200, got %q", m["http.status_code"])
	}
	if m["latency"] != "42.5" {
		t.Errorf("expected 42.5, got %q", m["latency"])
	}
	if m["cached"] != "true" {
		t.Errorf("expected true, got %q", m["cached"])
	}
}

func TestResourceServiceName(t *testing.T) {
	r := Resource{
		Attributes: []KeyValue{
			{Key: "service.name", Value: AnyValue{StringValue: strPtr("my-service")}},
			{Key: "service.version", Value: AnyValue{StringValue: strPtr("1.0")}},
		},
	}
	if got := ResourceServiceName(r); got != "my-service" {
		t.Errorf("expected 'my-service', got %q", got)
	}

	// Missing service.name.
	r2 := Resource{Attributes: []KeyValue{{Key: "host.name", Value: AnyValue{StringValue: strPtr("myhost")}}}}
	if got := ResourceServiceName(r2); got != "unknown" {
		t.Errorf("expected 'unknown', got %q", got)
	}
}

func strPtr(s string) *string   { return &s }
func float64Ptr(f float64) *float64 { return &f }
func boolPtr(b bool) *bool      { return &b }
