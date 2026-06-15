package otlp

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// RegisterGRPC registers the OTLP Trace/Metrics/Logs collector services on the
// given gRPC server. They share the exact ingestion pipeline as the HTTP
// server: each proto request is rendered to OTLP-JSON via protojson, decoded
// into the server's existing request structs, and run through convertTraces /
// ingestMetrics / ingestLogs.
func (s *Server) RegisterGRPC(gs *grpc.Server) {
	coltracepb.RegisterTraceServiceServer(gs, &grpcTraceService{s: s})
	colmetricspb.RegisterMetricsServiceServer(gs, &grpcMetricsService{s: s})
	collogspb.RegisterLogsServiceServer(gs, &grpcLogsService{s: s})
}

// protoJSON uses integer enum values to match the OTLP/HTTP JSON shapes the
// existing structs already parse.
var protoJSON = protojson.MarshalOptions{UseEnumNumbers: true}

// protoToStruct marshals a proto message to OTLP-JSON, then decodes it into the
// server's existing request struct so all conversion logic is reused.
func protoToStruct(m proto.Message, dst any) error {
	b, err := protoJSON.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// b64ToHex converts a protojson-encoded bytes field (standard base64) to the
// hex string the OTLP/HTTP path produces, keeping trace/span IDs consistent
// across protocols. Unparseable values are returned unchanged.
func b64ToHex(s string) string {
	if s == "" {
		return ""
	}
	if raw, err := base64.StdEncoding.DecodeString(s); err == nil {
		return hex.EncodeToString(raw)
	}
	return s
}

type grpcTraceService struct {
	coltracepb.UnimplementedTraceServiceServer
	s *Server
}

func (g *grpcTraceService) Export(_ context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	var tr ExportTraceRequest
	if err := protoToStruct(req, &tr); err != nil {
		return nil, err
	}
	// protojson encodes the bytes trace/span IDs as base64; the HTTP path uses
	// hex. Normalize so both protocols store identical IDs.
	for i := range tr.ResourceSpans {
		for j := range tr.ResourceSpans[i].ScopeSpans {
			for k := range tr.ResourceSpans[i].ScopeSpans[j].Spans {
				sp := &tr.ResourceSpans[i].ScopeSpans[j].Spans[k]
				sp.TraceID = b64ToHex(sp.TraceID)
				sp.SpanID = b64ToHex(sp.SpanID)
				sp.ParentSpanID = b64ToHex(sp.ParentSpanID)
			}
		}
	}
	g.s.ingestTraces(tr)
	return &coltracepb.ExportTraceServiceResponse{}, nil
}

type grpcMetricsService struct {
	colmetricspb.UnimplementedMetricsServiceServer
	s *Server
}

func (g *grpcMetricsService) Export(_ context.Context, req *colmetricspb.ExportMetricsServiceRequest) (*colmetricspb.ExportMetricsServiceResponse, error) {
	var mr ExportMetricsRequest
	if err := protoToStruct(req, &mr); err != nil {
		return nil, err
	}
	g.s.ingestMetrics(mr)
	return &colmetricspb.ExportMetricsServiceResponse{}, nil
}

type grpcLogsService struct {
	collogspb.UnimplementedLogsServiceServer
	s *Server
}

func (g *grpcLogsService) Export(_ context.Context, req *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	var lr ExportLogsRequest
	if err := protoToStruct(req, &lr); err != nil {
		return nil, err
	}
	for i := range lr.ResourceLogs {
		for j := range lr.ResourceLogs[i].ScopeLogs {
			for k := range lr.ResourceLogs[i].ScopeLogs[j].LogRecords {
				rec := &lr.ResourceLogs[i].ScopeLogs[j].LogRecords[k]
				rec.TraceID = b64ToHex(rec.TraceID)
				rec.SpanID = b64ToHex(rec.SpanID)
			}
		}
	}
	g.s.ingestLogs(lr)
	return &collogspb.ExportLogsServiceResponse{}, nil
}
