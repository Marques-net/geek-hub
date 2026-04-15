package matchcorev1

import proto "github.com/golang/protobuf/proto"

const _ = proto.ProtoPackageIsVersion4

type RoomRequest struct {
	GameType         string `protobuf:"bytes,1,opt,name=game_type,json=gameType,proto3" json:"game_type,omitempty"`
	RoomCode         string `protobuf:"bytes,2,opt,name=room_code,json=roomCode,proto3" json:"room_code,omitempty"`
	Nickname         string `protobuf:"bytes,3,opt,name=nickname,proto3" json:"nickname,omitempty"`
	PlayerToken      string `protobuf:"bytes,4,opt,name=player_token,json=playerToken,proto3" json:"player_token,omitempty"`
	SpectatorToken   string `protobuf:"bytes,5,opt,name=spectator_token,json=spectatorToken,proto3" json:"spectator_token,omitempty"`
	Mode             string `protobuf:"bytes,6,opt,name=mode,proto3" json:"mode,omitempty"`
	ClockControl     string `protobuf:"bytes,7,opt,name=clock_control,json=clockControl,proto3" json:"clock_control,omitempty"`
	ActionType       string `protobuf:"bytes,8,opt,name=action_type,json=actionType,proto3" json:"action_type,omitempty"`
	ActionPayloadJson string `protobuf:"bytes,9,opt,name=action_payload_json,json=actionPayloadJson,proto3" json:"action_payload_json,omitempty"`
}

func (x *RoomRequest) Reset() { *x = RoomRequest{} }
func (x *RoomRequest) String() string { return proto.CompactTextString(x) }
func (*RoomRequest) ProtoMessage() {}
func (x *RoomRequest) GetGameType() string { if x == nil { return "" }; return x.GameType }
func (x *RoomRequest) GetRoomCode() string { if x == nil { return "" }; return x.RoomCode }
func (x *RoomRequest) GetNickname() string { if x == nil { return "" }; return x.Nickname }
func (x *RoomRequest) GetPlayerToken() string { if x == nil { return "" }; return x.PlayerToken }
func (x *RoomRequest) GetSpectatorToken() string { if x == nil { return "" }; return x.SpectatorToken }
func (x *RoomRequest) GetMode() string { if x == nil { return "" }; return x.Mode }
func (x *RoomRequest) GetClockControl() string { if x == nil { return "" }; return x.ClockControl }
func (x *RoomRequest) GetActionType() string { if x == nil { return "" }; return x.ActionType }
func (x *RoomRequest) GetActionPayloadJson() string { if x == nil { return "" }; return x.ActionPayloadJson }

type RoomResponse struct {
	Ok           bool   `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	Code         string `protobuf:"bytes,2,opt,name=code,proto3" json:"code,omitempty"`
	Message      string `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	StatusCode   int32  `protobuf:"varint,4,opt,name=status_code,json=statusCode,proto3" json:"status_code,omitempty"`
	SnapshotJson string `protobuf:"bytes,5,opt,name=snapshot_json,json=snapshotJson,proto3" json:"snapshot_json,omitempty"`
	SessionJson  string `protobuf:"bytes,6,opt,name=session_json,json=sessionJson,proto3" json:"session_json,omitempty"`
	Left         bool   `protobuf:"varint,7,opt,name=left,proto3" json:"left,omitempty"`
}

func (x *RoomResponse) Reset() { *x = RoomResponse{} }
func (x *RoomResponse) String() string { return proto.CompactTextString(x) }
func (*RoomResponse) ProtoMessage() {}
func (x *RoomResponse) GetOk() bool { if x == nil { return false }; return x.Ok }
func (x *RoomResponse) GetCode() string { if x == nil { return "" }; return x.Code }
func (x *RoomResponse) GetMessage() string { if x == nil { return "" }; return x.Message }
func (x *RoomResponse) GetStatusCode() int32 { if x == nil { return 0 }; return x.StatusCode }
func (x *RoomResponse) GetSnapshotJson() string { if x == nil { return "" }; return x.SnapshotJson }
func (x *RoomResponse) GetSessionJson() string { if x == nil { return "" }; return x.SessionJson }
func (x *RoomResponse) GetLeft() bool { if x == nil { return false }; return x.Left }

type TickRequest struct{}

func (x *TickRequest) Reset() { *x = TickRequest{} }
func (x *TickRequest) String() string { return proto.CompactTextString(x) }
func (*TickRequest) ProtoMessage() {}

type TickResponse struct {
	SnapshotsJson []string `protobuf:"bytes,1,rep,name=snapshots_json,json=snapshotsJson,proto3" json:"snapshots_json,omitempty"`
}

func (x *TickResponse) Reset() { *x = TickResponse{} }
func (x *TickResponse) String() string { return proto.CompactTextString(x) }
func (*TickResponse) ProtoMessage() {}
func (x *TickResponse) GetSnapshotsJson() []string { if x == nil { return nil }; return x.SnapshotsJson }

func init() {
	proto.RegisterType((*RoomRequest)(nil), "games.matchcore.v1.RoomRequest")
	proto.RegisterType((*RoomResponse)(nil), "games.matchcore.v1.RoomResponse")
	proto.RegisterType((*TickRequest)(nil), "games.matchcore.v1.TickRequest")
	proto.RegisterType((*TickResponse)(nil), "games.matchcore.v1.TickResponse")
}
