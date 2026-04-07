// Package internal implements a native Pulumi provider for cloudmock using the
// gRPC resource provider protocol. This avoids any dependency on pulumi-terraform-bridge.
//
// The provider communicates directly with the cloudmock gateway using the same
// HTTP client approach as the Terraform provider, and generates its Pulumi schema
// dynamically from the cloudmock schema registry.
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	cmschema "github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/services/dynamodb"
	"github.com/Viridian-Inc/cloudmock/services/ec2"
	"github.com/Viridian-Inc/cloudmock/services/s3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// CloudmockProvider implements the Pulumi resource provider gRPC protocol.
type CloudmockProvider struct {
	UnimplementedResourceProviderServer

	endpoint  string
	region    string
	accessKey string
	secretKey string
	registry  *cmschema.Registry
	client    *apiClient
	schema    []byte // cached schema JSON
}

// NewProvider creates a new CloudmockProvider with the given schema registry.
func NewProvider(registry *cmschema.Registry) (*CloudmockProvider, error) {
	schemaJSON, err := GeneratePulumiSchemaJSON(registry)
	if err != nil {
		return nil, fmt.Errorf("generating Pulumi schema: %w", err)
	}

	return &CloudmockProvider{
		registry: registry,
		schema:   schemaJSON,
		// Defaults — overridden by Configure.
		endpoint:  "http://localhost:4566",
		region:    "us-east-1",
		accessKey: "test",
		secretKey: "test",
	}, nil
}

// DefaultRegistry builds the default schema registry from Tier 1 services.
func DefaultRegistry() *cmschema.Registry {
	s3Svc := s3.New()
	dynamoSvc := dynamodb.New("000000000000", "us-east-1")
	ec2Svc := ec2.New("000000000000", "us-east-1")

	var tier1 []cmschema.ResourceSchema
	for _, schemas := range [][]cmschema.ResourceSchema{
		s3Svc.ResourceSchemas(),
		dynamoSvc.ResourceSchemas(),
		ec2Svc.ResourceSchemas(),
	} {
		tier1 = append(tier1, schemas...)
	}

	return cmschema.BuildRegistry(tier1, nil)
}

// ── Pulumi Provider Protocol Implementation ─────────────────────────────────

// GetSchema returns the Pulumi package schema JSON.
func (p *CloudmockProvider) GetSchema(ctx context.Context, req *GetSchemaRequest) (*GetSchemaResponse, error) {
	return &GetSchemaResponse{
		Schema: string(p.schema),
	}, nil
}

// CheckConfig validates provider configuration.
func (p *CloudmockProvider) CheckConfig(ctx context.Context, req *CheckRequest) (*CheckResponse, error) {
	// Accept all config values — we apply defaults in Configure.
	return &CheckResponse{
		Inputs: req.News,
	}, nil
}

// DiffConfig computes differences in provider configuration.
func (p *CloudmockProvider) DiffConfig(ctx context.Context, req *DiffRequest) (*DiffResponse, error) {
	return &DiffResponse{}, nil
}

// Configure sets up the provider with the given configuration values.
func (p *CloudmockProvider) Configure(ctx context.Context, req *ConfigureRequest) (*ConfigureResponse, error) {
	if req.Args != nil {
		if v, ok := req.Args.Fields["endpoint"]; ok && v.GetStringValue() != "" {
			p.endpoint = v.GetStringValue()
		}
		if v, ok := req.Args.Fields["region"]; ok && v.GetStringValue() != "" {
			p.region = v.GetStringValue()
		}
		if v, ok := req.Args.Fields["accessKey"]; ok && v.GetStringValue() != "" {
			p.accessKey = v.GetStringValue()
		}
		if v, ok := req.Args.Fields["secretKey"]; ok && v.GetStringValue() != "" {
			p.secretKey = v.GetStringValue()
		}
	}

	p.client = newAPIClient(p.endpoint, p.region, p.accessKey, p.secretKey)

	return &ConfigureResponse{
		AcceptSecrets: true,
	}, nil
}

// Check validates resource inputs before create or update.
func (p *CloudmockProvider) Check(ctx context.Context, req *CheckRequest) (*CheckResponse, error) {
	rs, err := p.resolveResource(req.Urn)
	if err != nil {
		return nil, err
	}

	// Validate required fields.
	var failures []*CheckFailure
	if req.News != nil {
		for _, attr := range rs.Attributes {
			if !attr.Required {
				continue
			}
			v, ok := req.News.Fields[attr.Name]
			if !ok || isNullOrEmpty(v) {
				failures = append(failures, &CheckFailure{
					Property: attr.Name,
					Reason:   fmt.Sprintf("missing required property '%s'", attr.Name),
				})
			}
		}
	} else {
		for _, attr := range rs.Attributes {
			if attr.Required {
				failures = append(failures, &CheckFailure{
					Property: attr.Name,
					Reason:   fmt.Sprintf("missing required property '%s'", attr.Name),
				})
			}
		}
	}

	return &CheckResponse{
		Inputs:   req.News,
		Failures: failures,
	}, nil
}

// Diff computes what changed between old and new resource state.
func (p *CloudmockProvider) Diff(ctx context.Context, req *DiffRequest) (*DiffResponse, error) {
	rs, err := p.resolveResource(req.Urn)
	if err != nil {
		return nil, err
	}

	var diffs []string
	var replaces []string
	changes := DiffResponse_DIFF_NONE

	if req.Olds != nil && req.News != nil {
		for _, attr := range rs.Attributes {
			if attr.Computed && !attr.Required {
				continue
			}
			oldVal := req.Olds.Fields[attr.Name]
			newVal := req.News.Fields[attr.Name]

			if !structValuesEqual(oldVal, newVal) {
				diffs = append(diffs, attr.Name)
				changes = DiffResponse_DIFF_SOME
				if attr.ForceNew {
					replaces = append(replaces, attr.Name)
				}
			}
		}
	}

	// If no update action exists and there are diffs, everything requires replacement.
	if rs.UpdateAction == "" && len(diffs) > 0 {
		replaces = diffs
	}

	return &DiffResponse{
		Changes:  changes,
		Diffs:    diffs,
		Replaces: replaces,
	}, nil
}

// Create creates a new resource.
func (p *CloudmockProvider) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	if p.client == nil {
		p.client = newAPIClient(p.endpoint, p.region, p.accessKey, p.secretKey)
	}

	rs, err := p.resolveResource(req.Urn)
	if err != nil {
		return nil, err
	}

	inputs := structToMap(req.Properties)

	id, outputs, err := p.client.createResource(rs, inputs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	outputStruct, err := structpb.NewStruct(outputs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshaling outputs: %s", err)
	}

	return &CreateResponse{
		Id:         id,
		Properties: outputStruct,
	}, nil
}

// Read reads the current state of a resource.
func (p *CloudmockProvider) Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error) {
	if p.client == nil {
		p.client = newAPIClient(p.endpoint, p.region, p.accessKey, p.secretKey)
	}

	rs, err := p.resolveResource(req.Urn)
	if err != nil {
		return nil, err
	}

	result, err := p.client.readResource(rs, req.Id)
	if err != nil {
		if isNotFoundError(err) {
			// Resource no longer exists.
			return &ReadResponse{Id: ""}, nil
		}
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	if result == nil {
		// No read action defined — return existing state.
		return &ReadResponse{
			Id:         req.Id,
			Properties: req.Properties,
		}, nil
	}

	// Merge the read result with existing inputs.
	outputs := structToMap(req.Properties)
	mergeComputedAttrs(rs, outputs, result)

	outputStruct, err := structpb.NewStruct(outputs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshaling outputs: %s", err)
	}

	return &ReadResponse{
		Id:         req.Id,
		Properties: outputStruct,
		Inputs:     req.Inputs,
	}, nil
}

// Update updates an existing resource.
func (p *CloudmockProvider) Update(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error) {
	if p.client == nil {
		p.client = newAPIClient(p.endpoint, p.region, p.accessKey, p.secretKey)
	}

	rs, err := p.resolveResource(req.Urn)
	if err != nil {
		return nil, err
	}

	inputs := structToMap(req.News)

	result, err := p.client.updateResource(rs, req.Id, inputs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	outputs := make(map[string]any)
	for k, v := range inputs {
		outputs[k] = v
	}
	if result != nil {
		mergeComputedAttrs(rs, outputs, result)
	}

	outputStruct, err := structpb.NewStruct(outputs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshaling outputs: %s", err)
	}

	return &UpdateResponse{
		Properties: outputStruct,
	}, nil
}

// Delete deletes a resource.
func (p *CloudmockProvider) Delete(ctx context.Context, req *DeleteRequest) (*emptypb.Empty, error) {
	if p.client == nil {
		p.client = newAPIClient(p.endpoint, p.region, p.accessKey, p.secretKey)
	}

	rs, err := p.resolveResource(req.Urn)
	if err != nil {
		return nil, err
	}

	if err := p.client.deleteResource(rs, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	return &emptypb.Empty{}, nil
}

// GetPluginInfo returns information about this provider plugin.
func (p *CloudmockProvider) GetPluginInfo(ctx context.Context, req *emptypb.Empty) (*PluginInfo, error) {
	return &PluginInfo{
		Version: "0.1.0",
	}, nil
}

// Cancel signals the provider to gracefully shut down.
func (p *CloudmockProvider) Cancel(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// Invoke is not used but must be implemented.
func (p *CloudmockProvider) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Invoke is not supported")
}

// StreamInvoke is not used but must be implemented.
func (p *CloudmockProvider) StreamInvoke(req *InvokeRequest, server ResourceProvider_StreamInvokeServer) error {
	return status.Errorf(codes.Unimplemented, "StreamInvoke is not supported")
}

// Call is not used but must be implemented.
func (p *CloudmockProvider) Call(ctx context.Context, req *CallRequest) (*CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Call is not supported")
}

// Construct is not used but must be implemented.
func (p *CloudmockProvider) Construct(ctx context.Context, req *ConstructRequest) (*ConstructResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Construct is not supported")
}

// Attach is not used but must be implemented.
func (p *CloudmockProvider) Attach(ctx context.Context, req *PluginAttach) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// ── Server startup ──────────────────────────────────────────────────────────

// Serve starts the gRPC server on the given listener.
func (p *CloudmockProvider) Serve(lis net.Listener) error {
	srv := grpc.NewServer()
	RegisterResourceProviderServer(srv, p)
	return srv.Serve(lis)
}

// ServeMain is the main entry point for the provider binary.
// It follows the Pulumi provider protocol: listen on a port, print the port
// number to stdout so Pulumi can connect.
func ServeMain(registry *cmschema.Registry) {
	provider, err := NewProvider(registry)
	if err != nil {
		log.Fatalf("failed to create provider: %v", err)
	}

	// Listen on a random available port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Print the port number to stdout — this is how Pulumi discovers the provider.
	port := lis.Addr().(*net.TCPAddr).Port
	fmt.Fprintf(os.Stdout, "%d\n", port)

	srv := grpc.NewServer()
	RegisterResourceProviderServer(srv, provider)

	log.Printf("cloudmock provider listening on port %d", port)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}

// ── Internal helpers ────────────────────────────────────────────────────────

// resolveResource looks up the resource schema for a Pulumi URN.
// A URN looks like: urn:pulumi:stack::project::cloudmock:s3:Bucket::name
// The type token is the 4th segment: cloudmock:s3:Bucket
func (p *CloudmockProvider) resolveResource(urn string) (*cmschema.ResourceSchema, error) {
	token := extractTypeToken(urn)
	if token == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URN: %s", urn)
	}

	// Find the resource by matching its token against the registry.
	for _, rs := range p.registry.All() {
		if resourceToken(rs) == token {
			return &rs, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "unknown resource type: %s", token)
}

// extractTypeToken extracts the type token from a Pulumi URN.
// URN format: urn:pulumi:<stack>::<project>::<type>::<name>
func extractTypeToken(urn string) string {
	parts := strings.Split(urn, "::")
	if len(parts) < 4 {
		return ""
	}
	return parts[2]
}

// structToMap converts a protobuf Struct to a plain Go map.
func structToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	result := make(map[string]any)
	for k, v := range s.Fields {
		result[k] = valueToInterface(v)
	}
	return result
}

// valueToInterface converts a protobuf Value to a plain Go value.
func valueToInterface(v *structpb.Value) any {
	if v == nil {
		return nil
	}
	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return kind.NumberValue
	case *structpb.Value_StringValue:
		return kind.StringValue
	case *structpb.Value_BoolValue:
		return kind.BoolValue
	case *structpb.Value_StructValue:
		return structToMap(kind.StructValue)
	case *structpb.Value_ListValue:
		items := make([]any, len(kind.ListValue.Values))
		for i, item := range kind.ListValue.Values {
			items[i] = valueToInterface(item)
		}
		return items
	default:
		return nil
	}
}

// structValuesEqual compares two protobuf values for equality.
func structValuesEqual(a, b *structpb.Value) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Simple comparison by JSON serialization.
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

// isNullOrEmpty checks if a protobuf Value is null or an empty string.
func isNullOrEmpty(v *structpb.Value) bool {
	if v == nil {
		return true
	}
	if _, ok := v.Kind.(*structpb.Value_NullValue); ok {
		return true
	}
	if sv, ok := v.Kind.(*structpb.Value_StringValue); ok && sv.StringValue == "" {
		return true
	}
	return false
}
