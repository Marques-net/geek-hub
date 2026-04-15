package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	matchcorev1 "github.com/Marques-net/geek-hub/services/realtime-gateway/proto/matchcore"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MatchCoreClient struct {
	client  matchcorev1.MatchCoreServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewMatchCoreClient(cfg Config) (*MatchCoreClient, error) {
	conn, err := grpc.DialContext(
		context.Background(),
		cfg.MatchCoreAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, err
	}

	return &MatchCoreClient{
		client:  matchcorev1.NewMatchCoreServiceClient(conn),
		conn:    conn,
		timeout: time.Duration(cfg.MatchCoreTimeoutMs) * time.Millisecond,
	}, nil
}

func (c *MatchCoreClient) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *MatchCoreClient) Ready(ctx context.Context) error {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.Ready(callCtx, &matchcorev1.TickRequest{})
	})
	if err != nil {
		return err
	}
	if !response.GetOk() {
		return appErrorFromResponse(response, "Match core indisponível.")
	}

	return nil
}

func (c *MatchCoreClient) CreateRoom(ctx context.Context, payload CreateRoomPayload) (*RoomSnapshot, *SessionDescriptor, error) {
	slog.Info(
		"forwarding create_room to match-core",
		"game_type", payload.GameType,
		"nickname", payload.Nickname,
		"mode", payload.Mode,
		"clock_control", payload.ClockControl,
	)
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.CreateRoom(callCtx, &matchcorev1.RoomRequest{
			GameType:     string(payload.GameType),
			Nickname:     payload.Nickname,
			Mode:         string(payload.Mode),
			ClockControl: payload.ClockControl,
		})
	})
	if err != nil {
		return nil, nil, err
	}

	return parseSnapshotAndSession(response)
}

func (c *MatchCoreClient) JoinRoom(ctx context.Context, payload JoinRoomPayload) (*RoomSnapshot, *SessionDescriptor, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.JoinRoom(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			Nickname:       payload.Nickname,
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, nil, err
	}

	return parseSnapshotAndSession(response)
}

func (c *MatchCoreClient) LeaveRoom(ctx context.Context, payload LeaveRoomPayload) (*RoomSnapshot, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.LeaveRoom(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, err
	}

	return parseSnapshot(response)
}

func (c *MatchCoreClient) SyncState(ctx context.Context, payload SessionPayload) (*RoomSnapshot, *SessionDescriptor, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.SyncState(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, nil, err
	}

	snapshot, session, err := parseSnapshotAndOptionalSession(response)
	if err != nil {
		return nil, nil, err
	}
	if snapshot == nil {
		return nil, nil, &AppError{Message: "Resposta inválida do match core.", Code: "INVALID_MATCH_CORE_RESPONSE", StatusCode: 500}
	}

	return snapshot, session, nil
}

func (c *MatchCoreClient) SubmitAction(ctx context.Context, payload SubmitActionPayload) (*RoomSnapshot, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.SubmitAction(callCtx, &matchcorev1.RoomRequest{
			GameType:          string(payload.GameType),
			RoomCode:          strings.ToUpper(payload.RoomCode),
			PlayerToken:       payload.PlayerToken,
			ActionType:        payload.ActionType,
			ActionPayloadJson: payload.ActionPayloadJson,
		})
	})
	if err != nil {
		return nil, err
	}

	return requireSnapshot(response)
}

func (c *MatchCoreClient) Resign(ctx context.Context, payload SessionPayload) (*RoomSnapshot, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.Resign(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, err
	}

	return requireSnapshot(response)
}

func (c *MatchCoreClient) OfferDraw(ctx context.Context, payload SessionPayload) (*RoomSnapshot, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.OfferDraw(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, err
	}

	return requireSnapshot(response)
}

func (c *MatchCoreClient) AcceptDraw(ctx context.Context, payload SessionPayload) (*RoomSnapshot, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.AcceptDraw(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, err
	}

	return requireSnapshot(response)
}

func (c *MatchCoreClient) DeclineDraw(ctx context.Context, payload SessionPayload) (*RoomSnapshot, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.DeclineDraw(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, err
	}

	return requireSnapshot(response)
}

func (c *MatchCoreClient) MarkDisconnected(ctx context.Context, payload SessionPayload) (*RoomSnapshot, error) {
	response, err := c.invokeCommand(ctx, func(callCtx context.Context) (*matchcorev1.RoomResponse, error) {
		return c.client.MarkDisconnected(callCtx, &matchcorev1.RoomRequest{
			GameType:       string(payload.GameType),
			RoomCode:       strings.ToUpper(payload.RoomCode),
			PlayerToken:    payload.PlayerToken,
			SpectatorToken: payload.SpectatorToken,
		})
	})
	if err != nil {
		return nil, err
	}

	return parseSnapshot(response)
}

func (c *MatchCoreClient) TickActiveRooms(ctx context.Context) ([]RoomSnapshot, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.TickActiveRooms(callCtx, &matchcorev1.TickRequest{})
	if err != nil {
		return nil, err
	}

	snapshots := make([]RoomSnapshot, 0, len(response.GetSnapshotsJson()))
	for _, entry := range response.GetSnapshotsJson() {
		var snapshot RoomSnapshot
		if err := json.Unmarshal([]byte(entry), &snapshot); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

func (c *MatchCoreClient) invokeCommand(
	ctx context.Context,
	call func(context.Context) (*matchcorev1.RoomResponse, error),
) (*matchcorev1.RoomResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return call(callCtx)
}

func parseSnapshotAndSession(response *matchcorev1.RoomResponse) (*RoomSnapshot, *SessionDescriptor, error) {
	snapshot, err := requireSnapshot(response)
	if err != nil {
		return nil, nil, err
	}

	session, err := parseSession(response)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, &AppError{Message: "Resposta inválida do match core.", Code: "INVALID_MATCH_CORE_RESPONSE", StatusCode: 500}
	}

	return snapshot, session, nil
}

func parseSnapshotAndOptionalSession(response *matchcorev1.RoomResponse) (*RoomSnapshot, *SessionDescriptor, error) {
	if !response.GetOk() {
		return nil, nil, appErrorFromResponse(response, "Erro ao sincronizar estado.")
	}

	snapshot, err := parseSnapshot(response)
	if err != nil {
		return nil, nil, err
	}

	session, err := parseSession(response)
	if err != nil {
		return nil, nil, err
	}

	return snapshot, session, nil
}

func parseSnapshot(response *matchcorev1.RoomResponse) (*RoomSnapshot, error) {
	if !response.GetOk() {
		return nil, appErrorFromResponse(response, "Erro ao processar operação.")
	}
	if response.GetSnapshotJson() == "" {
		return nil, nil
	}

	var snapshot RoomSnapshot
	if err := json.Unmarshal([]byte(response.GetSnapshotJson()), &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func requireSnapshot(response *matchcorev1.RoomResponse) (*RoomSnapshot, error) {
	snapshot, err := parseSnapshot(response)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, &AppError{Message: "Resposta inválida do match core.", Code: "INVALID_MATCH_CORE_RESPONSE", StatusCode: 500}
	}
	return snapshot, nil
}

func parseSession(response *matchcorev1.RoomResponse) (*SessionDescriptor, error) {
	if response.GetSessionJson() == "" {
		return nil, nil
	}

	var session SessionDescriptor
	if err := json.Unmarshal([]byte(response.GetSessionJson()), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func appErrorFromResponse(response *matchcorev1.RoomResponse, fallback string) error {
	message := strings.TrimSpace(response.GetMessage())
	if message == "" {
		message = fallback
	}

	statusCode := int(response.GetStatusCode())
	if statusCode == 0 {
		statusCode = 500
	}

	return &AppError{
		Message:    message,
		Code:       response.GetCode(),
		StatusCode: statusCode,
	}
}
