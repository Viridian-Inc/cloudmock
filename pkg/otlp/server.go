package otlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/eventbus"
)

// Server handles OTLP/HTTP ingestion for traces, metrics, and logs.
type Server struct {
	dp  *dataplane.DataPlane
	bus *eventbus.Bus
	mux *http.ServeMux
	log *slog.Logger

	region    string
	accountID string
}

// NewServer creates a new OTLP HTTP server.
func NewServer(dp *dataplane.DataPlane, bus *eventbus.Bus, region, accountID string) *Server {
	s := &Server{
		dp:        dp,
		bus:       bus,
		mux:       http.NewServeMux(),
		log:       slog.Default().With("component", "otlp"),
		region:    region,
		accountID: accountID,
	}

	s.mux.HandleFunc("/v1/traces", s.handleTraces)
	s.mux.HandleFunc("/v1/metrics", s.handleMetrics)
	s.mux.HandleFunc("/v1/logs", s.handleLogs)
	s.mux.HandleFunc("/", s.handleRoot)

	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	// Health check or discovery endpoint.
	if r.URL.Path == "/" || r.URL.Path == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"cloudmock-otlp"}`))
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if rejected := s.rejectProtobuf(w, r); rejected {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MB limit
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req ExportTraceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	spans, requestEntries := s.convertTraces(req)

	ctx := context.Background()
	if s.dp.TraceW != nil && len(spans) > 0 {
		if err := s.dp.TraceW.WriteSpans(ctx, spans); err != nil {
			s.log.Error("failed to write spans", "error", err, "count", len(spans))
		}
	}

	// Write as request entries too so they show up in the request log.
	if s.dp.RequestW != nil {
		for _, entry := range requestEntries {
			if err := s.dp.RequestW.Write(ctx, entry); err != nil {
				s.log.Error("failed to write request entry", "error", err)
			}
		}
	}

	// Record metrics for each span.
	if s.dp.MetricW != nil {
		for _, sp := range spans {
			latencyMs := float64(sp.DurationNs) / 1e6
			_ = s.dp.MetricW.Record(ctx, sp.Service, sp.Action, latencyMs, sp.StatusCode)
		}
	}

	// Publish events for each trace to the event bus.
	if s.bus != nil {
		for _, sp := range spans {
			s.bus.Publish(&eventbus.Event{
				Source: "otlp",
				Type:   "otlp:TraceIngested:" + sp.Service,
				Detail: map[string]any{
					"traceId":  sp.TraceID,
					"spanId":   sp.SpanID,
					"service":  sp.Service,
					"action":   sp.Action,
					"duration": sp.DurationNs,
				},
				Time:      time.Now().UTC(),
				Region:    s.region,
				AccountID: s.accountID,
			})
		}
	}

	s.log.Info("ingested traces", "spans", len(spans))
	s.writeExportResponse(w)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if rejected := s.rejectProtobuf(w, r); rejected {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req ExportMetricsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	count := s.ingestMetrics(req)

	s.log.Info("ingested metrics", "data_points", count)
	s.writeExportResponse(w)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if rejected := s.rejectProtobuf(w, r); rejected {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req ExportLogsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	count := s.ingestLogs(req)

	s.log.Info("ingested logs", "records", count)
	s.writeExportResponse(w)
}

// ── Conversion helpers ──────────────────────────────────────────────────────

// convertTraces maps OTLP ResourceSpans to CloudMock dataplane.Span and RequestEntry.
func (s *Server) convertTraces(req ExportTraceRequest) ([]*dataplane.Span, []dataplane.RequestEntry) {
	var spans []*dataplane.Span
	var entries []dataplane.RequestEntry

	for _, rs := range req.ResourceSpans {
		serviceName := ResourceServiceName(rs.Resource)
		resourceAttrs := AttributeMap(rs.Resource.Attributes)

		for _, ss := range rs.ScopeSpans {
			for _, sp := range ss.Spans {
				startNano := parseNano(sp.StartTimeUnixNano)
				endNano := parseNano(sp.EndTimeUnixNano)
				startTime := time.Unix(0, startNano)
				endTime := time.Unix(0, endNano)
				durationNs := uint64(0)
				if endNano > startNano {
					durationNs = uint64(endNano - startNano)
				}

				spanAttrs := AttributeMap(sp.Attributes)

				// Merge resource attributes into span metadata.
				metadata := make(map[string]string, len(resourceAttrs)+len(spanAttrs))
				for k, v := range resourceAttrs {
					metadata[k] = v
				}
				for k, v := range spanAttrs {
					metadata[k] = v
				}

				// Derive HTTP method/path from span attributes if available.
				method := spanAttrs["http.method"]
				if method == "" {
					method = spanAttrs["http.request.method"]
				}
				path := spanAttrs["http.target"]
				if path == "" {
					path = spanAttrs["url.path"]
				}

				statusCode := 200
				errMsg := ""
				if sp.Status.Code == 2 { // ERROR
					statusCode = 500
					errMsg = sp.Status.Message
				}
				// Use HTTP status code from attributes if available.
				if httpStatus := spanAttrs["http.status_code"]; httpStatus != "" {
					if code, err := strconv.Atoi(httpStatus); err == nil {
						statusCode = code
					}
				}
				if httpStatus := spanAttrs["http.response.status_code"]; httpStatus != "" {
					if code, err := strconv.Atoi(httpStatus); err == nil {
						statusCode = code
					}
				}

				dpSpan := &dataplane.Span{
					TraceID:      sp.TraceID,
					SpanID:       sp.SpanID,
					ParentSpanID: sp.ParentSpanID,
					Service:      serviceName,
					Action:       sp.Name,
					Method:       method,
					Path:         path,
					StartTime:    startTime,
					EndTime:      endTime,
					DurationNs:   durationNs,
					StatusCode:   statusCode,
					Error:        errMsg,
					Metadata:     metadata,
				}
				spans = append(spans, dpSpan)

				// Also create a request entry for the request log.
				entry := dataplane.RequestEntry{
					ID:         uuid.New().String(),
					TraceID:    sp.TraceID,
					SpanID:     sp.SpanID,
					Timestamp:  startTime,
					Service:    serviceName,
					Action:     sp.Name,
					Method:     method,
					Path:       path,
					StatusCode: statusCode,
					Latency:    time.Duration(durationNs),
					LatencyMs:  float64(durationNs) / 1e6,
					Error:      errMsg,
					Level:      "app",
				}
				entries = append(entries, entry)
			}
		}
	}

	return spans, entries
}

// ingestMetrics maps OTLP metrics to CloudMock's metric store. Returns the number of data points ingested.
func (s *Server) ingestMetrics(req ExportMetricsRequest) int {
	if s.dp.MetricW == nil {
		return 0
	}

	ctx := context.Background()
	count := 0

	for _, rm := range req.ResourceMetrics {
		serviceName := ResourceServiceName(rm.Resource)

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				// Record each data point as a metric sample.
				if m.Gauge != nil {
					for _, dp := range m.Gauge.DataPoints {
						val := dataPointValue(dp)
						_ = s.dp.MetricW.Record(ctx, serviceName, m.Name, val, 200)
						count++
					}
				}
				if m.Sum != nil {
					for _, dp := range m.Sum.DataPoints {
						val := dataPointValue(dp)
						_ = s.dp.MetricW.Record(ctx, serviceName, m.Name, val, 200)
						count++
					}
				}
				if m.Histogram != nil {
					for _, dp := range m.Histogram.DataPoints {
						var val float64
						if dp.Sum != nil {
							val = *dp.Sum
						}
						_ = s.dp.MetricW.Record(ctx, serviceName, m.Name, val, 200)
						count++
					}
				}

				// Publish metric event.
				if s.bus != nil {
					s.bus.Publish(&eventbus.Event{
						Source: "otlp",
						Type:   "otlp:MetricIngested:" + serviceName,
						Detail: map[string]any{
							"service":    serviceName,
							"metricName": m.Name,
						},
						Time:      time.Now().UTC(),
						Region:    s.region,
						AccountID: s.accountID,
					})
				}
			}
		}
	}

	return count
}

// ingestLogs maps OTLP log records to CloudMock's request log. Returns the number of records ingested.
func (s *Server) ingestLogs(req ExportLogsRequest) int {
	if s.dp.RequestW == nil {
		return 0
	}

	ctx := context.Background()
	count := 0

	for _, rl := range req.ResourceLogs {
		serviceName := ResourceServiceName(rl.Resource)

		for _, sl := range rl.ScopeLogs {
			for _, lr := range sl.LogRecords {
				ts := time.Unix(0, parseNano(lr.TimeUnixNano))
				if ts.IsZero() {
					ts = time.Unix(0, parseNano(lr.ObservedTimeUnixNano))
				}
				if ts.IsZero() {
					ts = time.Now().UTC()
				}

				level := severityToLevel(lr.SeverityText, lr.SeverityNumber)
				message := lr.Body.StringVal()
				attrs := AttributeMap(lr.Attributes)

				errMsg := ""
				if level == "ERROR" || level == "FATAL" {
					errMsg = message
				}

				statusCode := 200
				if errMsg != "" {
					statusCode = 500
				}

				entry := dataplane.RequestEntry{
					ID:        uuid.New().String(),
					TraceID:   lr.TraceID,
					SpanID:    lr.SpanID,
					Timestamp: ts,
					Service:   serviceName,
					Action:    "log:" + level,
					Level:     "app",
					Error:     errMsg,
					StatusCode: statusCode,
					RequestHeaders: attrs,
					RequestBody:    message,
				}
				if err := s.dp.RequestW.Write(ctx, entry); err != nil {
					s.log.Error("failed to write log entry", "error", err)
				}
				count++

				// Publish log event.
				if s.bus != nil {
					s.bus.Publish(&eventbus.Event{
						Source: "otlp",
						Type:   "otlp:LogIngested:" + serviceName,
						Detail: map[string]any{
							"service":  serviceName,
							"severity": level,
							"message":  truncate(message, 200),
							"traceId":  lr.TraceID,
						},
						Time:      time.Now().UTC(),
						Region:    s.region,
						AccountID: s.accountID,
					})
				}
			}
		}
	}

	return count
}

// ── Response helpers ────────────────────────────────────────────────────────

// rejectProtobuf returns true and writes a 415 error if the request uses protobuf content type.
func (s *Server) rejectProtobuf(w http.ResponseWriter, r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/x-protobuf") || strings.Contains(ct, "application/protobuf") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_, _ = w.Write([]byte(`{"error":"protobuf encoding is not supported yet. Use JSON encoding by setting OTEL_EXPORTER_OTLP_PROTOCOL=http/json"}`))
		return true
	}
	return false
}

// writeExportResponse writes a successful OTLP export response.
func (s *Server) writeExportResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// OTLP spec: empty JSON object = full success (no partial rejection).
	_, _ = w.Write([]byte(`{}`))
}

// writeError writes an error response.
func (s *Server) writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = w.Write(resp)
}

// ── Utility functions ───────────────────────────────────────────────────────

// parseNano parses a nanosecond timestamp string to int64.
func parseNano(s string) int64 {
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// dataPointValue extracts the numeric value from a NumberDataPoint.
func dataPointValue(dp NumberDataPoint) float64 {
	if dp.AsDouble != nil {
		return *dp.AsDouble
	}
	if dp.AsInt != nil {
		v, _ := strconv.ParseInt(*dp.AsInt, 10, 64)
		return float64(v)
	}
	return 0
}

// severityToLevel maps OTLP severity to a CloudMock log level string.
func severityToLevel(text string, number int) string {
	if text != "" {
		upper := strings.ToUpper(text)
		switch {
		case strings.HasPrefix(upper, "FATAL"):
			return "FATAL"
		case strings.HasPrefix(upper, "ERROR"):
			return "ERROR"
		case strings.HasPrefix(upper, "WARN"):
			return "WARN"
		case strings.HasPrefix(upper, "INFO"):
			return "INFO"
		case strings.HasPrefix(upper, "DEBUG"):
			return "DEBUG"
		case strings.HasPrefix(upper, "TRACE"):
			return "TRACE"
		default:
			return upper
		}
	}

	// Fall back to severity number ranges per OTLP spec.
	switch {
	case number >= 21:
		return "FATAL"
	case number >= 17:
		return "ERROR"
	case number >= 13:
		return "WARN"
	case number >= 9:
		return "INFO"
	case number >= 5:
		return "DEBUG"
	case number >= 1:
		return "TRACE"
	default:
		return "INFO"
	}
}

// truncate returns s truncated to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... (%d more)", len(s)-maxLen)
}
