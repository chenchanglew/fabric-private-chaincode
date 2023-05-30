// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: tle.proto

package tlegrpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	TleService_GetMeta_FullMethodName    = "/TleService/getMeta"
	TleService_GetSession_FullMethodName = "/TleService/getSession"
)

// TleServiceClient is the client API for TleService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TleServiceClient interface {
	GetMeta(ctx context.Context, in *MetaRequest, opts ...grpc.CallOption) (*MetaResponse, error)
	GetSession(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*MetaResponse, error)
}

type tleServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewTleServiceClient(cc grpc.ClientConnInterface) TleServiceClient {
	return &tleServiceClient{cc}
}

func (c *tleServiceClient) GetMeta(ctx context.Context, in *MetaRequest, opts ...grpc.CallOption) (*MetaResponse, error) {
	out := new(MetaResponse)
	err := c.cc.Invoke(ctx, TleService_GetMeta_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tleServiceClient) GetSession(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*MetaResponse, error) {
	out := new(MetaResponse)
	err := c.cc.Invoke(ctx, TleService_GetSession_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TleServiceServer is the server API for TleService service.
// All implementations must embed UnimplementedTleServiceServer
// for forward compatibility
type TleServiceServer interface {
	GetMeta(context.Context, *MetaRequest) (*MetaResponse, error)
	GetSession(context.Context, *Empty) (*MetaResponse, error)
	mustEmbedUnimplementedTleServiceServer()
}

// UnimplementedTleServiceServer must be embedded to have forward compatible implementations.
type UnimplementedTleServiceServer struct {
}

func (UnimplementedTleServiceServer) GetMeta(context.Context, *MetaRequest) (*MetaResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMeta not implemented")
}
func (UnimplementedTleServiceServer) GetSession(context.Context, *Empty) (*MetaResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSession not implemented")
}
func (UnimplementedTleServiceServer) mustEmbedUnimplementedTleServiceServer() {}

// UnsafeTleServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TleServiceServer will
// result in compilation errors.
type UnsafeTleServiceServer interface {
	mustEmbedUnimplementedTleServiceServer()
}

func RegisterTleServiceServer(s grpc.ServiceRegistrar, srv TleServiceServer) {
	s.RegisterService(&TleService_ServiceDesc, srv)
}

func _TleService_GetMeta_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MetaRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TleServiceServer).GetMeta(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TleService_GetMeta_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TleServiceServer).GetMeta(ctx, req.(*MetaRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TleService_GetSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TleServiceServer).GetSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TleService_GetSession_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TleServiceServer).GetSession(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// TleService_ServiceDesc is the grpc.ServiceDesc for TleService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TleService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "TleService",
	HandlerType: (*TleServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "getMeta",
			Handler:    _TleService_GetMeta_Handler,
		},
		{
			MethodName: "getSession",
			Handler:    _TleService_GetSession_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "tle.proto",
}
