// Package internal contains hand-written Go types matching the Pulumi Resource Provider
// gRPC service protocol. This avoids pulling in github.com/pulumi/pulumi/sdk/v3
// which has heavy transitive dependencies.
//
// The types here implement the subset of pulumirpc.ResourceProvider that a basic
// CRUD provider needs. They are wire-compatible with the real Pulumi protobuf
// definitions and can be used with google.golang.org/grpc directly.
package internal

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// ── Request / Response types ────────────────────────────────────────────────

// GetSchemaRequest is the request for GetSchema.
type GetSchemaRequest struct {
	Version int32 `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
}

// GetSchemaResponse is the response for GetSchema.
type GetSchemaResponse struct {
	Schema string `protobuf:"bytes,1,opt,name=schema,proto3" json:"schema,omitempty"`
}

// ConfigureRequest is the request for Configure.
type ConfigureRequest struct {
	Variables    map[string]string `protobuf:"bytes,1,rep,name=variables,proto3" json:"variables,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Args         *structpb.Struct  `protobuf:"bytes,2,opt,name=args,proto3" json:"args,omitempty"`
	AcceptSecrets bool             `protobuf:"varint,3,opt,name=acceptSecrets,proto3" json:"acceptSecrets,omitempty"`
}

// ConfigureResponse is the response for Configure.
type ConfigureResponse struct {
	AcceptSecrets   bool `protobuf:"varint,1,opt,name=acceptSecrets,proto3" json:"acceptSecrets,omitempty"`
	SupportsPreview bool `protobuf:"varint,2,opt,name=supportsPreview,proto3" json:"supportsPreview,omitempty"`
	AcceptResources bool `protobuf:"varint,3,opt,name=acceptResources,proto3" json:"acceptResources,omitempty"`
	AcceptOutputs   bool `protobuf:"varint,4,opt,name=acceptOutputs,proto3" json:"acceptOutputs,omitempty"`
}

// CheckRequest is the request for Check and CheckConfig.
type CheckRequest struct {
	Urn        string           `protobuf:"bytes,1,opt,name=urn,proto3" json:"urn,omitempty"`
	Olds       *structpb.Struct `protobuf:"bytes,2,opt,name=olds,proto3" json:"olds,omitempty"`
	News       *structpb.Struct `protobuf:"bytes,3,opt,name=news,proto3" json:"news,omitempty"`
	RandomSeed []byte           `protobuf:"bytes,5,opt,name=randomSeed,proto3" json:"randomSeed,omitempty"`
}

// CheckResponse is the response for Check and CheckConfig.
type CheckResponse struct {
	Inputs   *structpb.Struct `protobuf:"bytes,1,opt,name=inputs,proto3" json:"inputs,omitempty"`
	Failures []*CheckFailure  `protobuf:"bytes,2,rep,name=failures,proto3" json:"failures,omitempty"`
}

// CheckFailure describes a single validation failure.
type CheckFailure struct {
	Property string `protobuf:"bytes,1,opt,name=property,proto3" json:"property,omitempty"`
	Reason   string `protobuf:"bytes,2,opt,name=reason,proto3" json:"reason,omitempty"`
}

// DiffRequest is the request for Diff and DiffConfig.
type DiffRequest struct {
	Id            string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Urn           string           `protobuf:"bytes,2,opt,name=urn,proto3" json:"urn,omitempty"`
	Olds          *structpb.Struct `protobuf:"bytes,3,opt,name=olds,proto3" json:"olds,omitempty"`
	News          *structpb.Struct `protobuf:"bytes,4,opt,name=news,proto3" json:"news,omitempty"`
	IgnoreChanges []string         `protobuf:"bytes,5,rep,name=ignoreChanges,proto3" json:"ignoreChanges,omitempty"`
}

// DiffResponse_DiffChanges enumerates the kind of changes a diff can report.
type DiffResponse_DiffChanges int32

const (
	DiffResponse_DIFF_UNKNOWN DiffResponse_DiffChanges = 0
	DiffResponse_DIFF_NONE    DiffResponse_DiffChanges = 1
	DiffResponse_DIFF_SOME    DiffResponse_DiffChanges = 2
)

// DiffResponse is the response for Diff and DiffConfig.
type DiffResponse struct {
	Replaces            []string                 `protobuf:"bytes,1,rep,name=replaces,proto3" json:"replaces,omitempty"`
	Stables             []string                 `protobuf:"bytes,2,rep,name=stables,proto3" json:"stables,omitempty"`
	DeleteBeforeReplace bool                     `protobuf:"varint,3,opt,name=deleteBeforeReplace,proto3" json:"deleteBeforeReplace,omitempty"`
	Changes             DiffResponse_DiffChanges `protobuf:"varint,4,opt,name=changes,proto3,enum=pulumirpc.DiffResponse_DiffChanges" json:"changes,omitempty"`
	Diffs               []string                 `protobuf:"bytes,5,rep,name=diffs,proto3" json:"diffs,omitempty"`
}

// CreateRequest is the request for Create.
type CreateRequest struct {
	Urn        string           `protobuf:"bytes,1,opt,name=urn,proto3" json:"urn,omitempty"`
	Properties *structpb.Struct `protobuf:"bytes,2,opt,name=properties,proto3" json:"properties,omitempty"`
	Timeout    float64          `protobuf:"fixed64,3,opt,name=timeout,proto3" json:"timeout,omitempty"`
	Preview    bool             `protobuf:"varint,4,opt,name=preview,proto3" json:"preview,omitempty"`
}

// CreateResponse is the response for Create.
type CreateResponse struct {
	Id         string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Properties *structpb.Struct `protobuf:"bytes,2,opt,name=properties,proto3" json:"properties,omitempty"`
}

// ReadRequest is the request for Read.
type ReadRequest struct {
	Id         string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Urn        string           `protobuf:"bytes,2,opt,name=urn,proto3" json:"urn,omitempty"`
	Properties *structpb.Struct `protobuf:"bytes,3,opt,name=properties,proto3" json:"properties,omitempty"`
	Inputs     *structpb.Struct `protobuf:"bytes,4,opt,name=inputs,proto3" json:"inputs,omitempty"`
}

// ReadResponse is the response for Read.
type ReadResponse struct {
	Id         string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Properties *structpb.Struct `protobuf:"bytes,2,opt,name=properties,proto3" json:"properties,omitempty"`
	Inputs     *structpb.Struct `protobuf:"bytes,3,opt,name=inputs,proto3" json:"inputs,omitempty"`
}

// UpdateRequest is the request for Update.
type UpdateRequest struct {
	Id            string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Urn           string           `protobuf:"bytes,2,opt,name=urn,proto3" json:"urn,omitempty"`
	Olds          *structpb.Struct `protobuf:"bytes,3,opt,name=olds,proto3" json:"olds,omitempty"`
	News          *structpb.Struct `protobuf:"bytes,4,opt,name=news,proto3" json:"news,omitempty"`
	Timeout       float64          `protobuf:"fixed64,5,opt,name=timeout,proto3" json:"timeout,omitempty"`
	IgnoreChanges []string         `protobuf:"bytes,6,rep,name=ignoreChanges,proto3" json:"ignoreChanges,omitempty"`
	Preview       bool             `protobuf:"varint,7,opt,name=preview,proto3" json:"preview,omitempty"`
}

// UpdateResponse is the response for Update.
type UpdateResponse struct {
	Properties *structpb.Struct `protobuf:"bytes,1,opt,name=properties,proto3" json:"properties,omitempty"`
}

// DeleteRequest is the request for Delete.
type DeleteRequest struct {
	Id         string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Urn        string           `protobuf:"bytes,2,opt,name=urn,proto3" json:"urn,omitempty"`
	Properties *structpb.Struct `protobuf:"bytes,3,opt,name=properties,proto3" json:"properties,omitempty"`
	Timeout    float64          `protobuf:"fixed64,4,opt,name=timeout,proto3" json:"timeout,omitempty"`
}

// InvokeRequest is the request for Invoke.
type InvokeRequest struct {
	Tok  string           `protobuf:"bytes,1,opt,name=tok,proto3" json:"tok,omitempty"`
	Args *structpb.Struct `protobuf:"bytes,2,opt,name=args,proto3" json:"args,omitempty"`
}

// InvokeResponse is the response for Invoke.
type InvokeResponse struct {
	Return   *structpb.Struct `protobuf:"bytes,1,opt,name=return,proto3" json:"return,omitempty"`
	Failures []*CheckFailure  `protobuf:"bytes,2,rep,name=failures,proto3" json:"failures,omitempty"`
}

// CallRequest is the request for Call.
type CallRequest struct {
	Tok  string           `protobuf:"bytes,1,opt,name=tok,proto3" json:"tok,omitempty"`
	Args *structpb.Struct `protobuf:"bytes,2,opt,name=args,proto3" json:"args,omitempty"`
}

// CallResponse is the response for Call.
type CallResponse struct {
	Return   *structpb.Struct `protobuf:"bytes,1,opt,name=return,proto3" json:"return,omitempty"`
	Failures []*CheckFailure  `protobuf:"bytes,3,rep,name=failures,proto3" json:"failures,omitempty"`
}

// ConstructRequest is the request for Construct.
type ConstructRequest struct {
	Project string           `protobuf:"bytes,1,opt,name=project,proto3" json:"project,omitempty"`
	Stack   string           `protobuf:"bytes,2,opt,name=stack,proto3" json:"stack,omitempty"`
	Type    string           `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	Name    string           `protobuf:"bytes,4,opt,name=name,proto3" json:"name,omitempty"`
	Inputs  *structpb.Struct `protobuf:"bytes,5,opt,name=inputs,proto3" json:"inputs,omitempty"`
}

// ConstructResponse is the response for Construct.
type ConstructResponse struct {
	Urn   string           `protobuf:"bytes,1,opt,name=urn,proto3" json:"urn,omitempty"`
	State *structpb.Struct `protobuf:"bytes,2,opt,name=state,proto3" json:"state,omitempty"`
}

// PluginInfo describes the provider plugin.
type PluginInfo struct {
	Version string `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
}

// PluginAttach is the request for Attach.
type PluginAttach struct {
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

// ── gRPC Service Definition ─────────────────────────────────────────────────

// ResourceProviderServer is the server API for the ResourceProvider service.
type ResourceProviderServer interface {
	GetSchema(context.Context, *GetSchemaRequest) (*GetSchemaResponse, error)
	CheckConfig(context.Context, *CheckRequest) (*CheckResponse, error)
	DiffConfig(context.Context, *DiffRequest) (*DiffResponse, error)
	Configure(context.Context, *ConfigureRequest) (*ConfigureResponse, error)
	Invoke(context.Context, *InvokeRequest) (*InvokeResponse, error)
	StreamInvoke(*InvokeRequest, ResourceProvider_StreamInvokeServer) error
	Call(context.Context, *CallRequest) (*CallResponse, error)
	Check(context.Context, *CheckRequest) (*CheckResponse, error)
	Diff(context.Context, *DiffRequest) (*DiffResponse, error)
	Create(context.Context, *CreateRequest) (*CreateResponse, error)
	Read(context.Context, *ReadRequest) (*ReadResponse, error)
	Update(context.Context, *UpdateRequest) (*UpdateResponse, error)
	Delete(context.Context, *DeleteRequest) (*emptypb.Empty, error)
	Construct(context.Context, *ConstructRequest) (*ConstructResponse, error)
	Cancel(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	GetPluginInfo(context.Context, *emptypb.Empty) (*PluginInfo, error)
	Attach(context.Context, *PluginAttach) (*emptypb.Empty, error)
}

// ResourceProvider_StreamInvokeServer is the server streaming interface for StreamInvoke.
type ResourceProvider_StreamInvokeServer interface {
	Send(*InvokeResponse) error
	grpc.ServerStream
}

type resourceProviderStreamInvokeServer struct {
	grpc.ServerStream
}

func (x *resourceProviderStreamInvokeServer) Send(m *InvokeResponse) error {
	return x.ServerStream.SendMsg(m)
}

// UnimplementedResourceProviderServer provides default unimplemented methods.
type UnimplementedResourceProviderServer struct{}

func (*UnimplementedResourceProviderServer) GetSchema(context.Context, *GetSchemaRequest) (*GetSchemaResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSchema not implemented")
}
func (*UnimplementedResourceProviderServer) CheckConfig(context.Context, *CheckRequest) (*CheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckConfig not implemented")
}
func (*UnimplementedResourceProviderServer) DiffConfig(context.Context, *DiffRequest) (*DiffResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DiffConfig not implemented")
}
func (*UnimplementedResourceProviderServer) Configure(context.Context, *ConfigureRequest) (*ConfigureResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Configure not implemented")
}
func (*UnimplementedResourceProviderServer) Invoke(context.Context, *InvokeRequest) (*InvokeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Invoke not implemented")
}
func (*UnimplementedResourceProviderServer) StreamInvoke(*InvokeRequest, ResourceProvider_StreamInvokeServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamInvoke not implemented")
}
func (*UnimplementedResourceProviderServer) Call(context.Context, *CallRequest) (*CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Call not implemented")
}
func (*UnimplementedResourceProviderServer) Check(context.Context, *CheckRequest) (*CheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Check not implemented")
}
func (*UnimplementedResourceProviderServer) Diff(context.Context, *DiffRequest) (*DiffResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Diff not implemented")
}
func (*UnimplementedResourceProviderServer) Create(context.Context, *CreateRequest) (*CreateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (*UnimplementedResourceProviderServer) Read(context.Context, *ReadRequest) (*ReadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Read not implemented")
}
func (*UnimplementedResourceProviderServer) Update(context.Context, *UpdateRequest) (*UpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Update not implemented")
}
func (*UnimplementedResourceProviderServer) Delete(context.Context, *DeleteRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (*UnimplementedResourceProviderServer) Construct(context.Context, *ConstructRequest) (*ConstructResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Construct not implemented")
}
func (*UnimplementedResourceProviderServer) Cancel(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Cancel not implemented")
}
func (*UnimplementedResourceProviderServer) GetPluginInfo(context.Context, *emptypb.Empty) (*PluginInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPluginInfo not implemented")
}
func (*UnimplementedResourceProviderServer) Attach(context.Context, *PluginAttach) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Attach not implemented")
}

// RegisterResourceProviderServer registers the ResourceProvider gRPC service.
func RegisterResourceProviderServer(s *grpc.Server, srv ResourceProviderServer) {
	s.RegisterService(&_ResourceProvider_serviceDesc, srv)
}

// ── gRPC service descriptor ─────────────────────────────────────────────────
// This is the hand-written equivalent of the generated service descriptor from
// pulumi's protobuf definitions.

var _ResourceProvider_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pulumirpc.ResourceProvider",
	HandlerType: (*ResourceProviderServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetSchema", Handler: _ResourceProvider_GetSchema_Handler},
		{MethodName: "CheckConfig", Handler: _ResourceProvider_CheckConfig_Handler},
		{MethodName: "DiffConfig", Handler: _ResourceProvider_DiffConfig_Handler},
		{MethodName: "Configure", Handler: _ResourceProvider_Configure_Handler},
		{MethodName: "Invoke", Handler: _ResourceProvider_Invoke_Handler},
		{MethodName: "Call", Handler: _ResourceProvider_Call_Handler},
		{MethodName: "Check", Handler: _ResourceProvider_Check_Handler},
		{MethodName: "Diff", Handler: _ResourceProvider_Diff_Handler},
		{MethodName: "Create", Handler: _ResourceProvider_Create_Handler},
		{MethodName: "Read", Handler: _ResourceProvider_Read_Handler},
		{MethodName: "Update", Handler: _ResourceProvider_Update_Handler},
		{MethodName: "Delete", Handler: _ResourceProvider_Delete_Handler},
		{MethodName: "Construct", Handler: _ResourceProvider_Construct_Handler},
		{MethodName: "Cancel", Handler: _ResourceProvider_Cancel_Handler},
		{MethodName: "GetPluginInfo", Handler: _ResourceProvider_GetPluginInfo_Handler},
		{MethodName: "Attach", Handler: _ResourceProvider_Attach_Handler},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamInvoke",
			Handler:       _ResourceProvider_StreamInvoke_Handler,
			ServerStreams:  true,
		},
	},
	Metadata: "pulumirpc/provider.proto",
}

// ── gRPC method handlers ────────────────────────────────────────────────────

func _ResourceProvider_GetSchema_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSchemaRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).GetSchema(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/GetSchema"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).GetSchema(ctx, req.(*GetSchemaRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_CheckConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).CheckConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/CheckConfig"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).CheckConfig(ctx, req.(*CheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_DiffConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiffRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).DiffConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/DiffConfig"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).DiffConfig(ctx, req.(*DiffRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Configure_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConfigureRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Configure(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Configure"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Configure(ctx, req.(*ConfigureRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Invoke_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InvokeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Invoke(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Invoke"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Invoke(ctx, req.(*InvokeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_StreamInvoke_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(InvokeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ResourceProviderServer).StreamInvoke(m, &resourceProviderStreamInvokeServer{stream})
}

func _ResourceProvider_Call_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CallRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Call(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Call"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Call(ctx, req.(*CallRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Check_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Check(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Check"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Check(ctx, req.(*CheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Diff_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiffRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Diff(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Diff"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Diff(ctx, req.(*DiffRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Create"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Create(ctx, req.(*CreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Read_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Read(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Read"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Read(ctx, req.(*ReadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Update"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Update(ctx, req.(*UpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Delete"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Construct_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConstructRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Construct(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Construct"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Construct(ctx, req.(*ConstructRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Cancel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Cancel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Cancel"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Cancel(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_GetPluginInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).GetPluginInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/GetPluginInfo"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).GetPluginInfo(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceProvider_Attach_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PluginAttach)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceProviderServer).Attach(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/pulumirpc.ResourceProvider/Attach"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceProviderServer).Attach(ctx, req.(*PluginAttach))
	}
	return interceptor(ctx, in, info, handler)
}
