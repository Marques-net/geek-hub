package strategyv1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const StrategyEngineService_GetAction_FullMethodName = "/games.strategy.v1.StrategyEngineService/GetAction"

type StrategyEngineServiceClient interface {
	GetAction(ctx context.Context, in *GetActionRequest, opts ...grpc.CallOption) (*GetActionResponse, error)
}

type strategyEngineServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStrategyEngineServiceClient(cc grpc.ClientConnInterface) StrategyEngineServiceClient {
	return &strategyEngineServiceClient{cc: cc}
}

func (c *strategyEngineServiceClient) GetAction(
	ctx context.Context,
	in *GetActionRequest,
	opts ...grpc.CallOption,
) (*GetActionResponse, error) {
	out := new(GetActionResponse)
	err := c.cc.Invoke(ctx, StrategyEngineService_GetAction_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type StrategyEngineServiceServer interface {
	GetAction(context.Context, *GetActionRequest) (*GetActionResponse, error)
}

type UnimplementedStrategyEngineServiceServer struct{}

func (UnimplementedStrategyEngineServiceServer) GetAction(
	context.Context,
	*GetActionRequest,
) (*GetActionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAction not implemented")
}

func RegisterStrategyEngineServiceServer(s grpc.ServiceRegistrar, srv StrategyEngineServiceServer) {
	s.RegisterService(&StrategyEngineService_ServiceDesc, srv)
}

func _StrategyEngineService_GetAction_Handler(
	srv any,
	ctx context.Context,
	dec func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	in := new(GetActionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}

	if interceptor == nil {
		return srv.(StrategyEngineServiceServer).GetAction(ctx, in)
	}

	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StrategyEngineService_GetAction_FullMethodName,
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(StrategyEngineServiceServer).GetAction(ctx, req.(*GetActionRequest))
	}

	return interceptor(ctx, in, info, handler)
}

var StrategyEngineService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "games.strategy.v1.StrategyEngineService",
	HandlerType: (*StrategyEngineServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetAction",
			Handler:    _StrategyEngineService_GetAction_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/strategy_engine.proto",
}
