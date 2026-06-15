package otlp

import (
	"context"
	"encoding/hex"
	"testing"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestGRPCTraceExport(t *testing.T) {
	srv, traceStore, _ := newTestServer()
	g := &grpcTraceService{s: srv}

	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := []byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8}

	req := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{{
			Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
				{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "grpc-svc"}}},
			}},
			ScopeSpans: []*tracepb.ScopeSpans{{
				Spans: []*tracepb.Span{{
					TraceId:           traceID,
					SpanId:            spanID,
					Name:              "GET /grpc",
					StartTimeUnixNano: 1711900000000000000,
					EndTimeUnixNano:   1711900001000000000,
					Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_OK},
				}},
			}},
		}},
	}

	if _, err := g.Export(context.Background(), req); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Stored under the hex-encoded trace id (base64→hex fixup of protojson output).
	trace := traceStore.Get(hex.EncodeToString(traceID))
	if trace == nil {
		t.Fatalf("span not stored under trace id %s", hex.EncodeToString(traceID))
	}
	if trace.Service != "grpc-svc" {
		t.Errorf("service = %q, want grpc-svc", trace.Service)
	}
	if trace.SpanID != hex.EncodeToString(spanID) {
		t.Errorf("spanID = %q, want %s", trace.SpanID, hex.EncodeToString(spanID))
	}
	if trace.Action != "GET /grpc" {
		t.Errorf("action = %q, want GET /grpc", trace.Action)
	}
}

func TestGRPCMetricsExport_NoError(t *testing.T) {
	srv, _, _ := newTestServer()
	g := &grpcMetricsService{s: srv}
	req := &colmetricspb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{{
			ScopeMetrics: []*metricspb.ScopeMetrics{{
				Metrics: []*metricspb.Metric{{Name: "http.server.duration"}},
			}},
		}},
	}
	if _, err := g.Export(context.Background(), req); err != nil {
		t.Fatalf("metrics Export: %v", err)
	}
}
