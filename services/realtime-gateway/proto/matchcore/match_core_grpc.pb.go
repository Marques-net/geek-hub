package matchcorev1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const (
	MatchCoreService_Ready_FullMethodName            = "/games.matchcore.v1.MatchCoreService/Ready"
	MatchCoreService_CreateRoom_FullMethodName       = "/games.matchcore.v1.MatchCoreService/CreateRoom"
	MatchCoreService_JoinRoom_FullMethodName         = "/games.matchcore.v1.MatchCoreService/JoinRoom"
	MatchCoreService_LeaveRoom_FullMethodName        = "/games.matchcore.v1.MatchCoreService/LeaveRoom"
	MatchCoreService_SyncState_FullMethodName        = "/games.matchcore.v1.MatchCoreService/SyncState"
	MatchCoreService_SubmitAction_FullMethodName     = "/games.matchcore.v1.MatchCoreService/SubmitAction"
	MatchCoreService_Resign_FullMethodName           = "/games.matchcore.v1.MatchCoreService/Resign"
	MatchCoreService_OfferDraw_FullMethodName        = "/games.matchcore.v1.MatchCoreService/OfferDraw"
	MatchCoreService_AcceptDraw_FullMethodName       = "/games.matchcore.v1.MatchCoreService/AcceptDraw"
	MatchCoreService_DeclineDraw_FullMethodName      = "/games.matchcore.v1.MatchCoreService/DeclineDraw"
	MatchCoreService_MarkDisconnected_FullMethodName = "/games.matchcore.v1.MatchCoreService/MarkDisconnected"
	MatchCoreService_TickActiveRooms_FullMethodName  = "/games.matchcore.v1.MatchCoreService/TickActiveRooms"
)

type MatchCoreServiceClient interface {
	Ready(ctx context.Context, in *TickRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	CreateRoom(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	JoinRoom(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	LeaveRoom(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	SyncState(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	SubmitAction(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	Resign(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	OfferDraw(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	AcceptDraw(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	DeclineDraw(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	MarkDisconnected(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error)
	TickActiveRooms(ctx context.Context, in *TickRequest, opts ...grpc.CallOption) (*TickResponse, error)
}

type matchCoreServiceClient struct{ cc grpc.ClientConnInterface }

func NewMatchCoreServiceClient(cc grpc.ClientConnInterface) MatchCoreServiceClient {
	return &matchCoreServiceClient{cc: cc}
}

func (c *matchCoreServiceClient) Ready(ctx context.Context, in *TickRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse)
	err := c.cc.Invoke(ctx, MatchCoreService_Ready_FullMethodName, in, out, opts...)
	if err != nil { return nil, err }
	return out, nil
}

func (c *matchCoreServiceClient) CreateRoom(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_CreateRoom_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) JoinRoom(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_JoinRoom_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) LeaveRoom(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_LeaveRoom_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) SyncState(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_SyncState_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) SubmitAction(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_SubmitAction_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) Resign(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_Resign_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) OfferDraw(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_OfferDraw_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) AcceptDraw(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_AcceptDraw_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) DeclineDraw(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_DeclineDraw_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) MarkDisconnected(ctx context.Context, in *RoomRequest, opts ...grpc.CallOption) (*RoomResponse, error) {
	out := new(RoomResponse); err := c.cc.Invoke(ctx, MatchCoreService_MarkDisconnected_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}
func (c *matchCoreServiceClient) TickActiveRooms(ctx context.Context, in *TickRequest, opts ...grpc.CallOption) (*TickResponse, error) {
	out := new(TickResponse); err := c.cc.Invoke(ctx, MatchCoreService_TickActiveRooms_FullMethodName, in, out, opts...); if err != nil { return nil, err }; return out, nil
}

type MatchCoreServiceServer interface {
	Ready(context.Context, *TickRequest) (*RoomResponse, error)
	CreateRoom(context.Context, *RoomRequest) (*RoomResponse, error)
	JoinRoom(context.Context, *RoomRequest) (*RoomResponse, error)
	LeaveRoom(context.Context, *RoomRequest) (*RoomResponse, error)
	SyncState(context.Context, *RoomRequest) (*RoomResponse, error)
	SubmitAction(context.Context, *RoomRequest) (*RoomResponse, error)
	Resign(context.Context, *RoomRequest) (*RoomResponse, error)
	OfferDraw(context.Context, *RoomRequest) (*RoomResponse, error)
	AcceptDraw(context.Context, *RoomRequest) (*RoomResponse, error)
	DeclineDraw(context.Context, *RoomRequest) (*RoomResponse, error)
	MarkDisconnected(context.Context, *RoomRequest) (*RoomResponse, error)
	TickActiveRooms(context.Context, *TickRequest) (*TickResponse, error)
}

type UnimplementedMatchCoreServiceServer struct{}

func (UnimplementedMatchCoreServiceServer) Ready(context.Context, *TickRequest) (*RoomResponse, error)            { return nil, status.Errorf(codes.Unimplemented, "method Ready not implemented") }
func (UnimplementedMatchCoreServiceServer) CreateRoom(context.Context, *RoomRequest) (*RoomResponse, error)       { return nil, status.Errorf(codes.Unimplemented, "method CreateRoom not implemented") }
func (UnimplementedMatchCoreServiceServer) JoinRoom(context.Context, *RoomRequest) (*RoomResponse, error)         { return nil, status.Errorf(codes.Unimplemented, "method JoinRoom not implemented") }
func (UnimplementedMatchCoreServiceServer) LeaveRoom(context.Context, *RoomRequest) (*RoomResponse, error)        { return nil, status.Errorf(codes.Unimplemented, "method LeaveRoom not implemented") }
func (UnimplementedMatchCoreServiceServer) SyncState(context.Context, *RoomRequest) (*RoomResponse, error)        { return nil, status.Errorf(codes.Unimplemented, "method SyncState not implemented") }
func (UnimplementedMatchCoreServiceServer) SubmitAction(context.Context, *RoomRequest) (*RoomResponse, error)     { return nil, status.Errorf(codes.Unimplemented, "method SubmitAction not implemented") }
func (UnimplementedMatchCoreServiceServer) Resign(context.Context, *RoomRequest) (*RoomResponse, error)           { return nil, status.Errorf(codes.Unimplemented, "method Resign not implemented") }
func (UnimplementedMatchCoreServiceServer) OfferDraw(context.Context, *RoomRequest) (*RoomResponse, error)        { return nil, status.Errorf(codes.Unimplemented, "method OfferDraw not implemented") }
func (UnimplementedMatchCoreServiceServer) AcceptDraw(context.Context, *RoomRequest) (*RoomResponse, error)       { return nil, status.Errorf(codes.Unimplemented, "method AcceptDraw not implemented") }
func (UnimplementedMatchCoreServiceServer) DeclineDraw(context.Context, *RoomRequest) (*RoomResponse, error)      { return nil, status.Errorf(codes.Unimplemented, "method DeclineDraw not implemented") }
func (UnimplementedMatchCoreServiceServer) MarkDisconnected(context.Context, *RoomRequest) (*RoomResponse, error) { return nil, status.Errorf(codes.Unimplemented, "method MarkDisconnected not implemented") }
func (UnimplementedMatchCoreServiceServer) TickActiveRooms(context.Context, *TickRequest) (*TickResponse, error)  { return nil, status.Errorf(codes.Unimplemented, "method TickActiveRooms not implemented") }

func RegisterMatchCoreServiceServer(s grpc.ServiceRegistrar, srv MatchCoreServiceServer) {
	s.RegisterService(&MatchCoreService_ServiceDesc, srv)
}

func unaryHandler[Req any, Resp any](srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor, method string, handler func(context.Context, *Req) (*Resp, error)) (any, error) {
	in := new(Req)
	if err := dec(in); err != nil { return nil, err }
	if interceptor == nil { return handler(ctx, in) }
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: method}
	wrapped := func(ctx context.Context, req any) (any, error) { return handler(ctx, req.(*Req)) }
	return interceptor(ctx, in, info, wrapped)
}

var MatchCoreService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "games.matchcore.v1.MatchCoreService",
	HandlerType: (*MatchCoreServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Ready", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[TickRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_Ready_FullMethodName, srv.(MatchCoreServiceServer).Ready)
		}},
		{MethodName: "CreateRoom", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_CreateRoom_FullMethodName, srv.(MatchCoreServiceServer).CreateRoom)
		}},
		{MethodName: "JoinRoom", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_JoinRoom_FullMethodName, srv.(MatchCoreServiceServer).JoinRoom)
		}},
		{MethodName: "LeaveRoom", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_LeaveRoom_FullMethodName, srv.(MatchCoreServiceServer).LeaveRoom)
		}},
		{MethodName: "SyncState", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_SyncState_FullMethodName, srv.(MatchCoreServiceServer).SyncState)
		}},
		{MethodName: "SubmitAction", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_SubmitAction_FullMethodName, srv.(MatchCoreServiceServer).SubmitAction)
		}},
		{MethodName: "Resign", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_Resign_FullMethodName, srv.(MatchCoreServiceServer).Resign)
		}},
		{MethodName: "OfferDraw", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_OfferDraw_FullMethodName, srv.(MatchCoreServiceServer).OfferDraw)
		}},
		{MethodName: "AcceptDraw", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_AcceptDraw_FullMethodName, srv.(MatchCoreServiceServer).AcceptDraw)
		}},
		{MethodName: "DeclineDraw", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_DeclineDraw_FullMethodName, srv.(MatchCoreServiceServer).DeclineDraw)
		}},
		{MethodName: "MarkDisconnected", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[RoomRequest, RoomResponse](srv, ctx, dec, interceptor, MatchCoreService_MarkDisconnected_FullMethodName, srv.(MatchCoreServiceServer).MarkDisconnected)
		}},
		{MethodName: "TickActiveRooms", Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			return unaryHandler[TickRequest, TickResponse](srv, ctx, dec, interceptor, MatchCoreService_TickActiveRooms_FullMethodName, srv.(MatchCoreServiceServer).TickActiveRooms)
		}},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/match_core.proto",
}
