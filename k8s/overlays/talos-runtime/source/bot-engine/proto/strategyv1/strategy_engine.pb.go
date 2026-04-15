package strategyv1

import proto "github.com/golang/protobuf/proto"

const _ = proto.ProtoPackageIsVersion4

type GetActionRequest struct {
	GameType      string   `protobuf:"bytes,1,opt,name=game_type,json=gameType,proto3" json:"game_type,omitempty"`
	RoomCode      string   `protobuf:"bytes,2,opt,name=room_code,json=roomCode,proto3" json:"room_code,omitempty"`
	GameId        string   `protobuf:"bytes,3,opt,name=game_id,json=gameId,proto3" json:"game_id,omitempty"`
	StateJson     string   `protobuf:"bytes,4,opt,name=state_json,json=stateJson,proto3" json:"state_json,omitempty"`
	Mode          string   `protobuf:"bytes,5,opt,name=mode,proto3" json:"mode,omitempty"`
	RecentActions []string `protobuf:"bytes,6,rep,name=recent_actions,json=recentActions,proto3" json:"recent_actions,omitempty"`
	MoveCount     uint32   `protobuf:"varint,7,opt,name=move_count,json=moveCount,proto3" json:"move_count,omitempty"`
}

func (x *GetActionRequest) Reset() { *x = GetActionRequest{} }
func (x *GetActionRequest) String() string { return proto.CompactTextString(x) }
func (*GetActionRequest) ProtoMessage() {}
func (x *GetActionRequest) GetGameType() string { if x == nil { return "" }; return x.GameType }
func (x *GetActionRequest) GetRoomCode() string { if x == nil { return "" }; return x.RoomCode }
func (x *GetActionRequest) GetGameId() string { if x == nil { return "" }; return x.GameId }
func (x *GetActionRequest) GetStateJson() string { if x == nil { return "" }; return x.StateJson }
func (x *GetActionRequest) GetMode() string { if x == nil { return "" }; return x.Mode }
func (x *GetActionRequest) GetRecentActions() []string { if x == nil { return nil }; return x.RecentActions }
func (x *GetActionRequest) GetActionCount() uint32 { if x == nil { return 0 }; return x.MoveCount }

type GetActionResponse struct {
	Found             bool   `protobuf:"varint,1,opt,name=found,proto3" json:"found,omitempty"`
	ActionType        string `protobuf:"bytes,2,opt,name=action_type,json=actionType,proto3" json:"action_type,omitempty"`
	ActionPayloadJson string `protobuf:"bytes,3,opt,name=action_payload_json,json=actionPayloadJson,proto3" json:"action_payload_json,omitempty"`
	CoachMessage      string `protobuf:"bytes,4,opt,name=coach_message,json=coachMessage,proto3" json:"coach_message,omitempty"`
}

func (x *GetActionResponse) Reset() { *x = GetActionResponse{} }
func (x *GetActionResponse) String() string { return proto.CompactTextString(x) }
func (*GetActionResponse) ProtoMessage() {}
func (x *GetActionResponse) GetFound() bool { if x == nil { return false }; return x.Found }
func (x *GetActionResponse) GetActionType() string { if x == nil { return "" }; return x.ActionType }
func (x *GetActionResponse) GetActionPayloadJson() string { if x == nil { return "" }; return x.ActionPayloadJson }
func (x *GetActionResponse) GetCoachMessage() string { if x == nil { return "" }; return x.CoachMessage }

func init() {
	proto.RegisterType((*GetActionRequest)(nil), "games.strategy.v1.GetActionRequest")
	proto.RegisterType((*GetActionResponse)(nil), "games.strategy.v1.GetActionResponse")
}
