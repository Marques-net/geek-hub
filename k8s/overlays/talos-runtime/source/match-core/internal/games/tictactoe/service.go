package tictactoe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	matchcorev1 "github.com/Marques-net/geek-hub/services/match-core/proto/matchcore"
	"github.com/google/uuid"
)

const (
	roomCodeChars         = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	botMoveDelayMs        = int64(900)
	easyBotNickname       = "Maquina (easy)"
	initialBoard          = "---------"
	actionTypeMove        = "move"
	actionTypeRestartGame = "restart_game"
)

var cellToIndex = map[string]int{
	"a1": 0,
	"b1": 1,
	"c1": 2,
	"a2": 3,
	"b2": 4,
	"c2": 5,
	"a3": 6,
	"b3": 7,
	"c3": 8,
}

var indexToCell = []string{
	"a1", "b1", "c1",
	"a2", "b2", "c2",
	"a3", "b3", "c3",
}

type winningLine struct {
	indexes [3]int
	reason  string
}

var winningLines = []winningLine{
	{indexes: [3]int{0, 1, 2}, reason: "row_top"},
	{indexes: [3]int{3, 4, 5}, reason: "row_middle"},
	{indexes: [3]int{6, 7, 8}, reason: "row_bottom"},
	{indexes: [3]int{0, 3, 6}, reason: "column_left"},
	{indexes: [3]int{1, 4, 7}, reason: "column_center"},
	{indexes: [3]int{2, 5, 8}, reason: "column_right"},
	{indexes: [3]int{0, 4, 8}, reason: "diagonal_main"},
	{indexes: [3]int{2, 4, 6}, reason: "diagonal_anti"},
}

type Service struct {
	config  Config
	store   *Store
	bot     *BotClient
	metrics *Metrics

	activeRoomsMu sync.Mutex
	activeRooms   map[string]struct{}
}

type moveActionPayload struct {
	Cell string `json:"cell"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

func NewService(config Config, store *Store, bot *BotClient, metrics *Metrics) *Service {
	return &Service{
		config:      config,
		store:       store,
		bot:         bot,
		metrics:     metrics,
		activeRooms: make(map[string]struct{}),
	}
}

func (s *Service) GameType() string {
	return GameTypeTicTacToe
}

func (s *Service) PrimeMetrics(ctx context.Context) error {
	rooms, err := s.store.ListRooms(ctx)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		s.trackRoom(room.RoomCode)
		if s.metrics != nil {
			s.metrics.UpdateRoom(room)
		}
	}

	return nil
}

func (s *Service) Ready(ctx context.Context) (*matchcorev1.RoomResponse, error) {
	if err := s.store.Ping(ctx); err != nil {
		return &matchcorev1.RoomResponse{
			Ok:         false,
			Code:       "REDIS_UNAVAILABLE",
			Message:    err.Error(),
			StatusCode: 503,
		}, nil
	}

	return &matchcorev1.RoomResponse{Ok: true}, nil
}

func (s *Service) CreateRoom(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	if req.GetGameType() != "" && req.GetGameType() != GameTypeTicTacToe {
		return errorResponse(newAppError("Tipo de jogo nao suportado neste runtime.", "UNSUPPORTED_GAME_TYPE", 400)), nil
	}

	nickname, appErr := normalizeNickname(req.GetNickname())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	mode, clockEnabled, appErr := parseCreateOptions(req.GetMode(), req.GetClockControl())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	roomCode, appErr := s.generateUniqueRoomCode(ctx)
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	timestamp := now()
	playerToken := uuid.NewString()
	initialClockMs := int64(0)
	if clockEnabled {
		initialClockMs = s.config.RoomClockMs()
	}

	state := &RoomState{
		GameID:       uuid.NewString(),
		GameType:     GameTypeTicTacToe,
		RoomCode:     roomCode,
		Mode:         mode,
		ClockEnabled: clockEnabled,
		FEN:          initialBoard,
		PGN:          "",
		Turn:         ColorWhite,
		Status:       GameStatusWaiting,
		White: &PlayerSeat{
			Nickname:    nickname,
			PlayerToken: playerToken,
			Connected:   true,
			JoinedAt:    timestamp,
			LastSeenAt:  timestamp,
		},
		Spectators: []SpectatorSession{},
		Clocks: GameClockState{
			InitialMs: initialClockMs,
			WhiteMs:   initialClockMs,
			BlackMs:   initialClockMs,
		},
		MoveHistory: []MoveRecord{},
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
		ExpiresAt:   timestamp + int64(s.config.GameTTLSeconds)*1000,
	}

	if mode == GameModeBotEasy {
		state.Black = s.createEasyBotSeat(timestamp)
		s.startGame(state, timestamp)
	}

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}

	session := &SessionDescriptor{
		RoomCode:    roomCode,
		GameType:    GameTypeTicTacToe,
		Role:        ViewerRolePlayer,
		Color:       colorPtr(ColorWhite),
		Nickname:    nickname,
		GameID:      state.GameID,
		Mode:        state.Mode,
		PlayerToken: playerToken,
	}

	return successResponse(s.toSnapshot(state), session, false)
}

func (s *Service) JoinRoom(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	nickname, appErr := normalizeNickname(req.GetNickname())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	state, appErr := s.requireRoom(ctx, req.GetRoomCode())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	timestamp := now()

	if req.GetPlayerToken() != "" {
		if color, seat := findSeatByToken(state, req.GetPlayerToken()); seat != nil {
			seat.Nickname = nickname
			seat.Connected = true
			seat.LastSeenAt = timestamp
			if err := s.saveRoom(ctx, state); err != nil {
				return nil, err
			}
			return successResponse(s.toSnapshot(state), &SessionDescriptor{
				RoomCode:    state.RoomCode,
				GameType:    state.GameType,
				Role:        ViewerRolePlayer,
				Color:       colorPtr(color),
				Nickname:    nickname,
				GameID:      state.GameID,
				Mode:        state.Mode,
				PlayerToken: seat.PlayerToken,
			}, false)
		}
	}

	if req.GetSpectatorToken() != "" {
		for index := range state.Spectators {
			spectator := &state.Spectators[index]
			if spectator.SpectatorToken != req.GetSpectatorToken() {
				continue
			}
			spectator.Nickname = nickname
			spectator.Connected = true
			spectator.LastSeenAt = timestamp
			if err := s.saveRoom(ctx, state); err != nil {
				return nil, err
			}
			return successResponse(s.toSnapshot(state), &SessionDescriptor{
				RoomCode:       state.RoomCode,
				GameType:       state.GameType,
				Role:           ViewerRoleSpectator,
				Nickname:       nickname,
				GameID:         state.GameID,
				Mode:           state.Mode,
				SpectatorToken: spectator.SpectatorToken,
			}, false)
		}
	}

	if state.White == nil {
		playerToken := uuid.NewString()
		state.White = &PlayerSeat{
			Nickname:    nickname,
			PlayerToken: playerToken,
			Connected:   true,
			JoinedAt:    timestamp,
			LastSeenAt:  timestamp,
		}
		if state.Black != nil && state.Status == GameStatusWaiting {
			s.startGame(state, timestamp)
		}
		if err := s.saveRoom(ctx, state); err != nil {
			return nil, err
		}
		return successResponse(s.toSnapshot(state), &SessionDescriptor{
			RoomCode:    state.RoomCode,
			GameType:    state.GameType,
			Role:        ViewerRolePlayer,
			Color:       colorPtr(ColorWhite),
			Nickname:    nickname,
			GameID:      state.GameID,
			Mode:        state.Mode,
			PlayerToken: playerToken,
		}, false)
	}

	if state.Black == nil {
		playerToken := uuid.NewString()
		state.Black = &PlayerSeat{
			Nickname:    nickname,
			PlayerToken: playerToken,
			Connected:   true,
			JoinedAt:    timestamp,
			LastSeenAt:  timestamp,
		}
		if state.Status == GameStatusWaiting {
			s.startGame(state, timestamp)
		}
		if err := s.saveRoom(ctx, state); err != nil {
			return nil, err
		}
		return successResponse(s.toSnapshot(state), &SessionDescriptor{
			RoomCode:    state.RoomCode,
			GameType:    state.GameType,
			Role:        ViewerRolePlayer,
			Color:       colorPtr(ColorBlack),
			Nickname:    nickname,
			GameID:      state.GameID,
			Mode:        state.Mode,
			PlayerToken: playerToken,
		}, false)
	}

	spectator := SpectatorSession{
		Nickname:       nickname,
		SpectatorToken: uuid.NewString(),
		Connected:      true,
		JoinedAt:       timestamp,
		LastSeenAt:     timestamp,
	}
	state.Spectators = append(state.Spectators, spectator)
	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}

	return successResponse(s.toSnapshot(state), &SessionDescriptor{
		RoomCode:       state.RoomCode,
		GameType:       state.GameType,
		Role:           ViewerRoleSpectator,
		Nickname:       nickname,
		GameID:         state.GameID,
		Mode:           state.Mode,
		SpectatorToken: spectator.SpectatorToken,
	}, false)
}

func (s *Service) LeaveRoom(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	state, appErr := s.requireRoom(ctx, req.GetRoomCode())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	timestamp := now()
	changed := false
	leavingColor := Color("")

	if req.GetPlayerToken() != "" {
		color, seat := findSeatByToken(state, req.GetPlayerToken())
		if seat == nil {
			return errorResponse(newAppError("Jogador nao encontrado nesta sala.", "SESSION_NOT_FOUND", 404)), nil
		}
		if seat.IsBot {
			return errorResponse(newAppError("A cadeira da maquina nao aceita comandos.", "BOT_CONTROL_FORBIDDEN", 403)), nil
		}
		leavingColor = color
		if state.Status == GameStatusWaiting || isFinishedStatus(state.Status) {
			if color == ColorWhite {
				state.White = nil
			} else {
				state.Black = nil
			}
		} else {
			seat.Connected = false
			seat.LastSeenAt = timestamp
		}
		changed = true
	}

	if req.GetSpectatorToken() != "" {
		filtered := make([]SpectatorSession, 0, len(state.Spectators))
		for _, spectator := range state.Spectators {
			if spectator.SpectatorToken == req.GetSpectatorToken() {
				changed = true
				continue
			}
			filtered = append(filtered, spectator)
		}
		state.Spectators = filtered
	}

	if !changed {
		return errorResponse(newAppError("Sessao nao encontrada para sair da sala.", "SESSION_NOT_FOUND", 404)), nil
	}

	if state.Mode == GameModeBotEasy && leavingColor != "" {
		closeBotRoomAfterHumanLeave(state, leavingColor, timestamp)
		_ = s.deleteRoom(ctx, state)
		return &matchcorev1.RoomResponse{Ok: true, Left: true}, nil
	}

	if state.White == nil && state.Black == nil && len(state.Spectators) == 0 {
		_ = s.deleteRoom(ctx, state)
		return &matchcorev1.RoomResponse{Ok: true, Left: true}, nil
	}

	if shouldDeleteFinishedRoom(state) {
		_ = s.deleteRoom(ctx, state)
		return &matchcorev1.RoomResponse{Ok: true, Left: true}, nil
	}

	hasHuman := false
	for _, seat := range []*PlayerSeat{state.White, state.Black} {
		if seat != nil && !seat.IsBot {
			hasHuman = true
			break
		}
	}

	if !hasHuman && len(state.Spectators) == 0 {
		_ = s.deleteRoom(ctx, state)
		return &matchcorev1.RoomResponse{Ok: true, Left: true}, nil
	}

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}
	return successResponse(s.toSnapshot(state), nil, true)
}

func (s *Service) SyncState(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	state, appErr := s.requireRoom(ctx, req.GetRoomCode())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	timestamp := now()
	var session *SessionDescriptor

	if req.GetPlayerToken() != "" {
		if color, seat := findSeatByToken(state, req.GetPlayerToken()); seat != nil {
			seat.Connected = true
			seat.LastSeenAt = timestamp
			session = &SessionDescriptor{
				RoomCode:    state.RoomCode,
				GameType:    state.GameType,
				Role:        ViewerRolePlayer,
				Color:       colorPtr(color),
				Nickname:    seat.Nickname,
				GameID:      state.GameID,
				Mode:        state.Mode,
				PlayerToken: seat.PlayerToken,
			}
		}
	}

	if session == nil && req.GetSpectatorToken() != "" {
		for index := range state.Spectators {
			spectator := &state.Spectators[index]
			if spectator.SpectatorToken != req.GetSpectatorToken() {
				continue
			}
			spectator.Connected = true
			spectator.LastSeenAt = timestamp
			session = &SessionDescriptor{
				RoomCode:       state.RoomCode,
				GameType:       state.GameType,
				Role:           ViewerRoleSpectator,
				Nickname:       spectator.Nickname,
				GameID:         state.GameID,
				Mode:           state.Mode,
				SpectatorToken: spectator.SpectatorToken,
			}
			break
		}
	}

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}
	return successResponse(s.toSnapshot(state), session, false)
}

func (s *Service) SubmitAction(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	switch req.GetActionType() {
	case actionTypeMove:
		return s.submitMove(ctx, req)
	case actionTypeRestartGame:
		return s.restartGame(ctx, req)
	default:
		return errorResponse(newAppError("Acao nao suportada para jogo da velha.", "UNSUPPORTED_ACTION", 400)), nil
	}
}

func (s *Service) submitMove(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	var payload moveActionPayload
	if err := json.Unmarshal([]byte(req.GetActionPayloadJson()), &payload); err != nil {
		return errorResponse(newAppError("A carga da acao e invalida.", "INVALID_ACTION_PAYLOAD", 400)), nil
	}

	cell := normalizeCell(firstNonEmpty(payload.Cell, payload.To, payload.From))
	if cell == "" {
		return errorResponse(newAppError("Informe a casa da jogada.", "INVALID_ACTION_PAYLOAD", 400)), nil
	}

	state, appErr := s.requireRoom(ctx, req.GetRoomCode())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	color, seat := findSeatByToken(state, req.GetPlayerToken())
	if seat == nil {
		return errorResponse(newAppError("Jogador nao encontrado nesta sala.", "SESSION_NOT_FOUND", 404)), nil
	}
	if seat.IsBot {
		return errorResponse(newAppError("A cadeira da maquina nao aceita comandos.", "BOT_CONTROL_FORBIDDEN", 403)), nil
	}

	timestamp := now()
	s.advanceClock(state, timestamp)
	s.finishIfTimedOut(state)
	if state.Status == GameStatusTimeout {
		if err := s.saveRoom(ctx, state); err != nil {
			return nil, err
		}
		return successResponse(s.toSnapshot(state), nil, false)
	}

	if state.Status != GameStatusActive {
		return errorResponse(newAppError("A partida nao esta ativa.", "GAME_NOT_ACTIVE", 400)), nil
	}
	if state.Turn != color {
		return errorResponse(newAppError("Nao e o seu turno.", "OUT_OF_TURN", 400)), nil
	}

	if appErr := s.applyMoveToState(state, cell, timestamp); appErr != nil {
		return errorResponse(appErr), nil
	}

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}
	return successResponse(s.toSnapshot(state), nil, false)
}

func (s *Service) restartGame(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	state, appErr := s.requireRoom(ctx, req.GetRoomCode())
	if appErr != nil {
		return errorResponse(appErr), nil
	}

	color, seat := findSeatByToken(state, req.GetPlayerToken())
	if seat == nil {
		return errorResponse(newAppError("Jogador nao encontrado nesta sala.", "SESSION_NOT_FOUND", 404)), nil
	}
	if seat.IsBot {
		return errorResponse(newAppError("A cadeira da maquina nao aceita comandos.", "BOT_CONTROL_FORBIDDEN", 403)), nil
	}
	if !isFinishedStatus(state.Status) {
		return errorResponse(newAppError("A partida so pode ser reiniciada depois de encerrar.", "GAME_NOT_FINISHED", 400)), nil
	}
	if state.White == nil || state.Black == nil {
		return errorResponse(newAppError("A sala precisa manter os dois jogadores para iniciar uma nova partida.", "RESTART_REQUIRES_PLAYERS", 400)), nil
	}

	timestamp := now()
	seat.Connected = true
	seat.LastSeenAt = timestamp
	s.resetRoomForRestart(state, timestamp)

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}

	return successResponse(s.toSnapshot(state), &SessionDescriptor{
		RoomCode:    state.RoomCode,
		GameType:    state.GameType,
		Role:        ViewerRolePlayer,
		Color:       colorPtr(color),
		Nickname:    seat.Nickname,
		GameID:      state.GameID,
		Mode:        state.Mode,
		PlayerToken: seat.PlayerToken,
	}, false)
}

func (s *Service) Resign(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	state, appErr := s.requireRoom(ctx, req.GetRoomCode())
	if appErr != nil {
		return errorResponse(appErr), nil
	}
	color, seat := findSeatByToken(state, req.GetPlayerToken())
	if seat == nil {
		return errorResponse(newAppError("Jogador nao encontrado nesta sala.", "SESSION_NOT_FOUND", 404)), nil
	}
	if seat.IsBot {
		return errorResponse(newAppError("A cadeira da maquina nao aceita comandos.", "BOT_CONTROL_FORBIDDEN", 403)), nil
	}
	if state.Status != GameStatusActive && state.Status != GameStatusWaiting {
		return errorResponse(newAppError("A partida ja foi encerrada.", "GAME_ALREADY_FINISHED", 400)), nil
	}

	winner := ColorWhite
	if color == ColorWhite {
		winner = ColorBlack
	}
	s.finishGame(state, &winner, GameStatusResigned, "resignation")

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}
	return successResponse(s.toSnapshot(state), nil, false)
}

func (s *Service) OfferDraw(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.drawAction(ctx, req, "offer")
}

func (s *Service) AcceptDraw(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.drawAction(ctx, req, "accept")
}

func (s *Service) DeclineDraw(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.drawAction(ctx, req, "decline")
}

func (s *Service) MarkDisconnected(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	state, err := s.store.GetRoom(ctx, normalizeRoomCode(req.GetRoomCode()))
	if err != nil {
		return nil, err
	}
	if state == nil {
		return &matchcorev1.RoomResponse{Ok: true}, nil
	}

	timestamp := now()
	changed := false
	if req.GetPlayerToken() != "" {
		_, seat := findSeatByToken(state, req.GetPlayerToken())
		if seat != nil && !seat.IsBot {
			seat.Connected = false
			seat.LastSeenAt = timestamp
			changed = true
		}
	}
	if req.GetSpectatorToken() != "" {
		for index := range state.Spectators {
			if state.Spectators[index].SpectatorToken != req.GetSpectatorToken() {
				continue
			}
			state.Spectators[index].Connected = false
			state.Spectators[index].LastSeenAt = timestamp
			changed = true
			break
		}
	}

	if !changed {
		return &matchcorev1.RoomResponse{Ok: true}, nil
	}

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}
	return successResponse(s.toSnapshot(state), nil, false)
}

func (s *Service) TickActiveRooms(ctx context.Context) (*matchcorev1.TickResponse, error) {
	roomCodes := s.listActiveRooms()
	snapshots := make([]string, 0)

	for _, roomCode := range roomCodes {
		state, err := s.store.GetRoom(ctx, roomCode)
		if err != nil {
			return nil, err
		}
		if state == nil {
			s.untrackRoom(roomCode)
			if s.metrics != nil {
				s.metrics.DeleteRoom(roomCode)
			}
			continue
		}

		timestamp := now()
		changed := false

		s.advanceClock(state, timestamp)
		beforeStatus := state.Status
		s.finishIfTimedOut(state)
		changed = changed || beforeStatus != state.Status

		if s.shouldPlayBotMove(state, timestamp) {
			if err := s.performBotMove(ctx, state, timestamp); err != nil {
				nextDue := timestamp + botMoveDelayMs
				state.BotMoveDueAt = &nextDue
				changed = true
			} else {
				changed = true
			}
		}

		if !changed {
			continue
		}

		if err := s.saveRoom(ctx, state); err != nil {
			return nil, err
		}

		payload, err := json.Marshal(s.toSnapshot(state))
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, string(payload))
	}

	return &matchcorev1.TickResponse{SnapshotsJson: snapshots}, nil
}

func (s *Service) drawAction(ctx context.Context, req *matchcorev1.RoomRequest, action string) (*matchcorev1.RoomResponse, error) {
	state, appErr := s.requireRoom(ctx, req.GetRoomCode())
	if appErr != nil {
		return errorResponse(appErr), nil
	}
	color, seat := findSeatByToken(state, req.GetPlayerToken())
	if seat == nil {
		return errorResponse(newAppError("Jogador nao encontrado nesta sala.", "SESSION_NOT_FOUND", 404)), nil
	}
	if seat.IsBot {
		return errorResponse(newAppError("A cadeira da maquina nao aceita comandos.", "BOT_CONTROL_FORBIDDEN", 403)), nil
	}
	if state.Mode == GameModeBotEasy {
		return errorResponse(newAppError("Empate nao esta disponivel contra a maquina easy.", "DRAW_UNAVAILABLE", 400)), nil
	}

	switch action {
	case "offer":
		if state.Status != GameStatusActive {
			return errorResponse(newAppError("So e possivel oferecer empate com a partida ativa.", "GAME_NOT_ACTIVE", 400)), nil
		}
		if state.DrawOffer != nil && state.DrawOffer.OfferedBy == color {
			return errorResponse(newAppError("Voce ja ofereceu empate.", "DRAW_ALREADY_OFFERED", 400)), nil
		}
		state.DrawOffer = &DrawOffer{OfferedBy: color, CreatedAt: now()}
	case "accept":
		if state.DrawOffer == nil || state.DrawOffer.OfferedBy == color {
			return errorResponse(newAppError("Nao existe oferta de empate pendente para aceitar.", "DRAW_NOT_PENDING", 400)), nil
		}
		s.finishGame(state, nil, GameStatusDraw, "draw_offer_accepted")
	case "decline":
		if state.DrawOffer == nil || state.DrawOffer.OfferedBy == color {
			return errorResponse(newAppError("Nao existe oferta de empate pendente para recusar.", "DRAW_NOT_PENDING", 400)), nil
		}
		state.DrawOffer = nil
	}

	if err := s.saveRoom(ctx, state); err != nil {
		return nil, err
	}
	return successResponse(s.toSnapshot(state), nil, false)
}

func (s *Service) saveRoom(ctx context.Context, state *RoomState) error {
	if err := s.store.SaveRoom(ctx, state); err != nil {
		return err
	}

	s.trackRoom(state.RoomCode)
	if s.metrics != nil {
		s.metrics.UpdateRoom(state)
	}

	return nil
}

func (s *Service) deleteRoom(ctx context.Context, state *RoomState) error {
	if state == nil {
		return nil
	}

	if s.metrics != nil {
		s.metrics.UpdateRoom(state)
	}

	if err := s.store.DeleteRoom(ctx, state.RoomCode); err != nil {
		return err
	}

	s.untrackRoom(state.RoomCode)
	if s.metrics != nil {
		s.metrics.DeleteRoom(state.RoomCode)
	}

	return nil
}

func (s *Service) performBotMove(ctx context.Context, state *RoomState, timestamp int64) error {
	move, err := s.bot.GetAction(ctx, state)
	if err != nil {
		return err
	}

	if move == nil {
		if isBoardFull(state.FEN) {
			s.finishGame(state, nil, GameStatusDraw, "board_filled")
			return nil
		}

		nextDue := timestamp + botMoveDelayMs
		state.BotMoveDueAt = &nextDue
		return nil
	}

	return s.applyMoveToState(state, normalizeCell(move.Cell), timestamp)
}

func (s *Service) applyMoveToState(state *RoomState, cell string, timestamp int64) *AppError {
	board, appErr := normalizedBoard(state.FEN)
	if appErr != nil {
		return appErr
	}

	index, ok := cellToIndex[cell]
	if !ok {
		return newAppError("Casa invalida.", "INVALID_MOVE", 400)
	}
	if board[index] != '-' {
		return newAppError("Esta casa ja esta ocupada.", "INVALID_MOVE", 400)
	}

	moveColor := state.Turn
	board[index] = markForColor(moveColor)
	nextBoard := string(board)
	record := MoveRecord{
		SAN:   formatMoveSAN(moveColor, cell),
		LAN:   strings.ToUpper(cell),
		From:  cell,
		To:    cell,
		Color: moveColor,
		FEN:   nextBoard,
		At:    timestamp,
	}

	state.FEN = nextBoard
	state.PGN = buildPGN(append(state.MoveHistory, record))
	state.LastMove = &record
	state.MoveHistory = append(state.MoveHistory, record)
	state.DrawOffer = nil

	if winner, endReason := winnerForBoard(nextBoard); winner != nil {
		s.finishGame(state, winner, GameStatusWon, endReason)
		return nil
	}

	if isBoardFull(nextBoard) {
		s.finishGame(state, nil, GameStatusDraw, "board_filled")
		return nil
	}

	state.Turn = oppositeColor(state.Turn)
	if state.ClockEnabled {
		state.Clocks.ActiveColor = colorPtr(state.Turn)
		state.Clocks.TurnStartedAt = int64Ptr(timestamp)
	} else {
		state.Clocks.ActiveColor = nil
		state.Clocks.TurnStartedAt = nil
	}
	if s.isBotTurn(state) {
		nextDue := timestamp + botMoveDelayMs
		state.BotMoveDueAt = &nextDue
	} else {
		state.BotMoveDueAt = nil
	}

	return nil
}

func (s *Service) requireRoom(ctx context.Context, roomCode string) (*RoomState, *AppError) {
	state, err := s.store.GetRoom(ctx, normalizeRoomCode(roomCode))
	if err != nil {
		return nil, newAppError(err.Error(), "STORE_ERROR", 500)
	}
	if state == nil {
		return nil, newAppError("Sala nao encontrada.", "ROOM_NOT_FOUND", 404)
	}
	return state, nil
}

func (s *Service) generateUniqueRoomCode(ctx context.Context) (string, *AppError) {
	for attempt := 0; attempt < 10; attempt++ {
		roomCode := createRoomCode()
		state, err := s.store.GetRoom(ctx, roomCode)
		if err != nil {
			return "", newAppError(err.Error(), "STORE_ERROR", 500)
		}
		if state == nil {
			return roomCode, nil
		}
	}
	return "", newAppError("Nao foi possivel gerar uma sala agora.", "ROOM_GENERATION_FAILED", 503)
}

func (s *Service) createEasyBotSeat(timestamp int64) *PlayerSeat {
	return &PlayerSeat{
		Nickname:    easyBotNickname,
		PlayerToken: uuid.NewString(),
		Connected:   true,
		JoinedAt:    timestamp,
		LastSeenAt:  timestamp,
		IsBot:       true,
		BotLevel:    BotLevelEasy,
	}
}

func (s *Service) startGame(state *RoomState, timestamp int64) {
	state.Status = GameStatusActive
	state.Turn = ColorWhite
	state.DrawOffer = nil
	if state.StartedAt == nil {
		state.StartedAt = int64Ptr(timestamp)
	}
	state.FinishedAt = nil
	state.Clocks.ActiveColor = nil
	state.Clocks.TurnStartedAt = nil
	state.BotMoveDueAt = nil
}

func (s *Service) resetRoomForRestart(state *RoomState, timestamp int64) {
	initialClockMs := int64(0)
	if state.ClockEnabled {
		initialClockMs = state.Clocks.InitialMs
		if initialClockMs <= 0 {
			initialClockMs = s.config.RoomClockMs()
		}
	}

	if state.White != nil && state.White.IsBot {
		state.White.Connected = true
		state.White.LastSeenAt = timestamp
	}
	if state.Black != nil && state.Black.IsBot {
		state.Black.Connected = true
		state.Black.LastSeenAt = timestamp
	}

	state.GameID = uuid.NewString()
	state.FEN = initialBoard
	state.PGN = ""
	state.Turn = ColorWhite
	state.Status = GameStatusActive
	state.Winner = nil
	state.EndReason = nil
	state.MoveHistory = []MoveRecord{}
	state.LastMove = nil
	state.DrawOffer = nil
	state.BotMoveDueAt = nil
	state.StartedAt = int64Ptr(timestamp)
	state.FinishedAt = nil
	state.Clocks = GameClockState{
		InitialMs: initialClockMs,
		WhiteMs:   initialClockMs,
		BlackMs:   initialClockMs,
	}
	state.UpdatedAt = timestamp
	state.ExpiresAt = timestamp + int64(s.config.GameTTLSeconds)*1000
}

func (s *Service) advanceClock(state *RoomState, timestamp int64) {
	if !state.ClockEnabled {
		return
	}
	if state.Clocks.ActiveColor == nil || state.Clocks.TurnStartedAt == nil {
		return
	}

	elapsed := timestamp - *state.Clocks.TurnStartedAt
	if elapsed <= 0 {
		return
	}

	if *state.Clocks.ActiveColor == ColorWhite {
		state.Clocks.WhiteMs = maxInt64(0, state.Clocks.WhiteMs-elapsed)
	} else {
		state.Clocks.BlackMs = maxInt64(0, state.Clocks.BlackMs-elapsed)
	}
	state.Clocks.TurnStartedAt = int64Ptr(timestamp)
}

func (s *Service) finishIfTimedOut(state *RoomState) {
	if !state.ClockEnabled || state.Status != GameStatusActive {
		return
	}
	if state.Clocks.WhiteMs <= 0 {
		winner := ColorBlack
		s.finishGame(state, &winner, GameStatusTimeout, "white_flag_fell")
		return
	}
	if state.Clocks.BlackMs <= 0 {
		winner := ColorWhite
		s.finishGame(state, &winner, GameStatusTimeout, "black_flag_fell")
	}
}

func (s *Service) finishGame(state *RoomState, winner *Color, status GameStatus, endReason string) {
	state.Status = status
	state.Winner = winner
	state.EndReason = stringPtr(endReason)
	if state.FinishedAt == nil {
		state.FinishedAt = int64Ptr(now())
	}
	state.DrawOffer = nil
	state.Clocks.ActiveColor = nil
	state.Clocks.TurnStartedAt = nil
	state.BotMoveDueAt = nil
}

func (s *Service) shouldPlayBotMove(state *RoomState, timestamp int64) bool {
	return state.Status == GameStatusActive && state.BotMoveDueAt != nil && *state.BotMoveDueAt <= timestamp && s.isBotTurn(state)
}

func (s *Service) isBotTurn(state *RoomState) bool {
	if state.Turn == ColorWhite {
		return state.White != nil && state.White.IsBot
	}
	return state.Black != nil && state.Black.IsBot
}

func (s *Service) toSnapshot(state *RoomState) *RoomSnapshot {
	viewerCount := len(state.Spectators)
	if state.White != nil {
		viewerCount++
	}
	if state.Black != nil {
		viewerCount++
	}

	return &RoomSnapshot{
		GameID:       state.GameID,
		GameType:     state.GameType,
		RoomCode:     state.RoomCode,
		Mode:         state.Mode,
		ClockEnabled: state.ClockEnabled,
		FEN:          state.FEN,
		PGN:          state.PGN,
		Turn:         state.Turn,
		Status:       state.Status,
		Winner:       state.Winner,
		EndReason:    state.EndReason,
		White:        seatSnapshot(state.White),
		Black:        seatSnapshot(state.Black),
		ViewerCount:  viewerCount,
		Clocks:       state.Clocks,
		MoveHistory:  state.MoveHistory,
		LastMove:     state.LastMove,
		DrawOffer:    state.DrawOffer,
		ServerNow:    now(),
	}
}

func (s *Service) trackRoom(roomCode string) {
	s.activeRoomsMu.Lock()
	defer s.activeRoomsMu.Unlock()
	s.activeRooms[roomCode] = struct{}{}
}

func (s *Service) untrackRoom(roomCode string) {
	s.activeRoomsMu.Lock()
	defer s.activeRoomsMu.Unlock()
	delete(s.activeRooms, roomCode)
}

func (s *Service) listActiveRooms() []string {
	s.activeRoomsMu.Lock()
	defer s.activeRoomsMu.Unlock()
	values := make([]string, 0, len(s.activeRooms))
	for roomCode := range s.activeRooms {
		values = append(values, roomCode)
	}
	return values
}

func successResponse(snapshot *RoomSnapshot, session *SessionDescriptor, left bool) (*matchcorev1.RoomResponse, error) {
	response := &matchcorev1.RoomResponse{Ok: true, Left: left}
	if snapshot != nil {
		payload, err := json.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
		response.SnapshotJson = string(payload)
	}
	if session != nil {
		payload, err := json.Marshal(session)
		if err != nil {
			return nil, err
		}
		response.SessionJson = string(payload)
	}
	return response, nil
}

func errorResponse(err *AppError) *matchcorev1.RoomResponse {
	return &matchcorev1.RoomResponse{
		Ok:         false,
		Code:       err.Code,
		Message:    err.Message,
		StatusCode: err.StatusCode,
	}
}

func normalizeNickname(value string) (string, *AppError) {
	nickname := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(nickname) < 2 || len(nickname) > 24 {
		return "", newAppError("Use um nickname entre 2 e 24 caracteres.", "INVALID_NICKNAME", 400)
	}
	return nickname, nil
}

func parseCreateOptions(modeValue string, clockValue string) (GameMode, bool, *AppError) {
	switch strings.TrimSpace(modeValue) {
	case "", string(GameModePVP):
		clockEnabled, appErr := parseClockControlDefault(clockValue)
		return GameModePVP, clockEnabled, appErr
	case "pvp_untimed":
		return GameModePVP, false, nil
	case string(GameModeBotEasy):
		clockEnabled, appErr := parseClockControlDefault(clockValue)
		return GameModeBotEasy, clockEnabled, appErr
	case "bot_easy_untimed":
		return GameModeBotEasy, false, nil
	default:
		return "", false, newAppError("Modo de jogo invalido.", "INVALID_MODE", 400)
	}
}

func parseClockControlDefault(value string) (bool, *AppError) {
	switch strings.TrimSpace(value) {
	case "", "timed":
		return true, nil
	case "untimed":
		return false, nil
	default:
		return false, newAppError("Controle de tempo invalido.", "INVALID_CLOCK_CONTROL", 400)
	}
}

func normalizeRoomCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func createRoomCode() string {
	buffer := make([]byte, 6)
	for index := range buffer {
		buffer[index] = roomCodeChars[rand.Intn(len(roomCodeChars))]
	}
	return string(buffer)
}

func normalizeCell(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func findSeatByToken(state *RoomState, playerToken string) (Color, *PlayerSeat) {
	if state.White != nil && state.White.PlayerToken == playerToken {
		return ColorWhite, state.White
	}
	if state.Black != nil && state.Black.PlayerToken == playerToken {
		return ColorBlack, state.Black
	}
	return "", nil
}

func seatSnapshot(seat *PlayerSeat) SeatSnapshot {
	if seat == nil {
		return SeatSnapshot{}
	}
	return SeatSnapshot{
		Nickname:  stringPtr(seat.Nickname),
		Connected: seat.Connected,
		IsBot:     seat.IsBot,
		BotLevel:  botLevelPtr(seat.BotLevel),
	}
}

func normalizedBoard(board string) ([]byte, *AppError) {
	trimmed := strings.TrimSpace(board)
	if trimmed == "" {
		trimmed = initialBoard
	}
	if len(trimmed) != len(indexToCell) {
		return nil, newAppError("Estado da partida invalido.", "INVALID_STATE", 500)
	}

	bytes := []byte(strings.ToLower(trimmed))
	for _, cell := range bytes {
		if cell != '-' && cell != 'x' && cell != 'o' {
			return nil, newAppError("Estado da partida invalido.", "INVALID_STATE", 500)
		}
	}

	return bytes, nil
}

func winnerForBoard(board string) (*Color, string) {
	if len(board) != len(indexToCell) {
		return nil, ""
	}

	for _, line := range winningLines {
		first := board[line.indexes[0]]
		if first == '-' {
			continue
		}
		if board[line.indexes[1]] == first && board[line.indexes[2]] == first {
			winner := colorFromMark(first)
			return &winner, line.reason
		}
	}

	return nil, ""
}

func isBoardFull(board string) bool {
	for _, cell := range board {
		if cell == '-' {
			return false
		}
	}
	return true
}

func buildPGN(history []MoveRecord) string {
	parts := make([]string, 0, len(history))
	moveNumber := 1
	for index, move := range history {
		if index%2 == 0 {
			parts = append(parts, fmt.Sprintf("%d. %s", moveNumber, move.SAN))
			moveNumber++
			continue
		}
		parts = append(parts, move.SAN)
	}
	return strings.Join(parts, " ")
}

func formatMoveSAN(color Color, cell string) string {
	return fmt.Sprintf("%s@%s", markLabel(color), strings.ToUpper(cell))
}

func markLabel(color Color) string {
	if color == ColorWhite {
		return "X"
	}
	return "O"
}

func markForColor(color Color) byte {
	if color == ColorWhite {
		return 'x'
	}
	return 'o'
}

func colorFromMark(mark byte) Color {
	if mark == 'x' {
		return ColorWhite
	}
	return ColorBlack
}

func newAppError(message, code string, statusCode int32) *AppError {
	return &AppError{Message: message, Code: code, StatusCode: statusCode}
}

func now() int64 { return time.Now().UnixMilli() }

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func colorPtr(value Color) *Color { return &value }

func botLevelPtr(value BotLevel) *BotLevel {
	if value == "" {
		return nil
	}
	return &value
}

func int64Ptr(value int64) *int64 { return &value }

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeStateDefaults(state *RoomState) {
	if state == nil {
		return
	}
	if state.GameType == "" {
		state.GameType = GameTypeTicTacToe
	}
	if state.Mode == "" {
		state.Mode = GameModePVP
	}
	if state.FEN == "" {
		state.FEN = initialBoard
	}
	if state.Turn == "" {
		if len(state.MoveHistory)%2 == 0 {
			state.Turn = ColorWhite
		} else {
			state.Turn = ColorBlack
		}
	}
	if state.StartedAt == nil {
		switch {
		case len(state.MoveHistory) > 0 && state.MoveHistory[0].At > 0:
			state.StartedAt = int64Ptr(state.MoveHistory[0].At)
		case state.Status != GameStatusWaiting && state.CreatedAt > 0:
			state.StartedAt = int64Ptr(state.CreatedAt)
		}
	}
	if isFinishedStatus(state.Status) {
		if state.FinishedAt == nil {
			switch {
			case state.UpdatedAt > 0:
				state.FinishedAt = int64Ptr(state.UpdatedAt)
			case state.LastMove != nil && state.LastMove.At > 0:
				state.FinishedAt = int64Ptr(state.LastMove.At)
			case state.StartedAt != nil:
				state.FinishedAt = int64Ptr(*state.StartedAt)
			case state.CreatedAt > 0:
				state.FinishedAt = int64Ptr(state.CreatedAt)
			}
		}
	} else {
		state.FinishedAt = nil
	}
	if !state.ClockEnabled && (state.Clocks.InitialMs > 0 || state.Clocks.WhiteMs > 0 || state.Clocks.BlackMs > 0) {
		state.ClockEnabled = true
	}
}

func isFinishedStatus(status GameStatus) bool {
	switch status {
	case GameStatusWon, GameStatusDraw, GameStatusResigned, GameStatusTimeout:
		return true
	default:
		return false
	}
}

func shouldDeleteFinishedRoom(state *RoomState) bool {
	return isFinishedStatus(state.Status) && humanSeatsCount(state) == 0
}

func closeBotRoomAfterHumanLeave(state *RoomState, leavingColor Color, timestamp int64) {
	if state == nil {
		return
	}

	if !isFinishedStatus(state.Status) {
		winner := oppositeColor(leavingColor)
		safelyFinishGameForRoomClosure(state, &winner, timestamp)
	}

	if state.White != nil {
		state.White.Connected = false
		state.White.LastSeenAt = timestamp
	}
	if state.Black != nil {
		state.Black.Connected = false
		state.Black.LastSeenAt = timestamp
	}
	state.Spectators = nil
	state.UpdatedAt = timestamp
}

func safelyFinishGameForRoomClosure(state *RoomState, winner *Color, timestamp int64) {
	state.Status = GameStatusResigned
	state.Winner = winner
	state.EndReason = stringPtr("player_left_room")
	if state.FinishedAt == nil {
		state.FinishedAt = int64Ptr(timestamp)
	}
	state.DrawOffer = nil
	state.Clocks.ActiveColor = nil
	state.Clocks.TurnStartedAt = nil
	state.BotMoveDueAt = nil
}

func oppositeColor(color Color) Color {
	if color == ColorWhite {
		return ColorBlack
	}
	return ColorWhite
}

func humanSeatsCount(state *RoomState) int {
	total := 0
	for _, seat := range []*PlayerSeat{state.White, state.Black} {
		if seat != nil && !seat.IsBot {
			total++
		}
	}
	return total
}
