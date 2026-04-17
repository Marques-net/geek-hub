package tictactoe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	strategyv1 "github.com/Marques-net/geek-hub/services/match-core/proto/strategyv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BotMove struct {
	Cell         string
	CoachMessage string
}

type BotClient struct {
	client strategyv1.StrategyEngineServiceClient
	conn   *grpc.ClientConn
	config Config
}

func NewBotClient(config Config) (*BotClient, error) {
	conn, err := grpc.DialContext(
		context.Background(),
		fmt.Sprintf("%s:%d", config.BotEngineHost, config.BotEnginePort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &BotClient{
		client: strategyv1.NewStrategyEngineServiceClient(conn),
		conn:   conn,
		config: config,
	}, nil
}

func (c *BotClient) Close() error {
	return c.conn.Close()
}

func (c *BotClient) GetAction(ctx context.Context, state *RoomState) (*BotMove, error) {
	callCtx, cancel := context.WithTimeout(ctx, time.Duration(c.config.BotEngineTimeoutMs)*time.Millisecond)
	defer cancel()

	statePayload, err := json.Marshal(map[string]any{
		"board": state.FEN,
		"turn":  state.Turn,
	})
	if err != nil {
		return nil, err
	}

	response, err := c.client.GetAction(callCtx, &strategyv1.GetActionRequest{
		GameType:      state.GameType,
		RoomCode:      state.RoomCode,
		GameId:        state.GameID,
		StateJson:     string(statePayload),
		Mode:          string(state.Mode),
		RecentActions: recentSans(state.MoveHistory, 9),
		MoveCount:     uint32(len(state.MoveHistory)),
	})
	if err != nil {
		return nil, err
	}

	if !response.GetFound() {
		return nil, nil
	}

	var actionPayload struct {
		Cell string `json:"cell"`
	}
	if err := json.Unmarshal([]byte(response.GetActionPayloadJson()), &actionPayload); err != nil {
		return nil, err
	}

	return &BotMove{
		Cell:         actionPayload.Cell,
		CoachMessage: response.GetCoachMessage(),
	}, nil
}

func recentSans(history []MoveRecord, size int) []string {
	if len(history) <= size {
		values := make([]string, 0, len(history))
		for _, move := range history {
			values = append(values, move.SAN)
		}
		return values
	}

	values := make([]string, 0, size)
	for _, move := range history[len(history)-size:] {
		values = append(values, move.SAN)
	}
	return values
}
