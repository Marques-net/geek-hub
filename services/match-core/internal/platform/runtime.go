package platform

import (
	"context"

	matchcorev1 "github.com/Marques-net/geek-hub/services/match-core/proto/matchcore"
)

type Runtime interface {
	GameType() string
	Ready(context.Context) (*matchcorev1.RoomResponse, error)
	CreateRoom(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	JoinRoom(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	LeaveRoom(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	SyncState(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	SubmitAction(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	Resign(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	OfferDraw(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	AcceptDraw(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	DeclineDraw(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	MarkDisconnected(context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)
	TickActiveRooms(context.Context) (*matchcorev1.TickResponse, error)
}

type Registry struct {
	runtimes map[string]Runtime
}

func NewRegistry(values ...Runtime) *Registry {
	runtimes := make(map[string]Runtime, len(values))
	for _, runtime := range values {
		if runtime == nil {
			continue
		}
		runtimes[runtime.GameType()] = runtime
	}
	return &Registry{runtimes: runtimes}
}

func (r *Registry) Resolve(gameType string) Runtime {
	if r == nil {
		return nil
	}
	return r.runtimes[gameType]
}

