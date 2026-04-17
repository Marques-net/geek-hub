package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	socketio "github.com/doquangtan/socketio/v4"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const maxChatHistory = 120

type roomChatState struct {
	mu               sync.Mutex
	history          []ChatMessage
	seeded           bool
	lastBotCommentAt int64
}

type Server struct {
	config     Config
	logger     *slog.Logger
	tracer     oteltrace.Tracer
	matchCore  *MatchCoreClient
	metrics    *Metrics
	io         *socketio.Io
	emitToRoom func(roomCode string, eventName string, args ...any)

	sessions  sync.Map
	chatRooms sync.Map
}

type operationResult struct {
	snapshot *RoomSnapshot
	roomCode string
	session  *SessionDescriptor
	extras   map[string]any
}

func NewServer(config Config, logger *slog.Logger, tracer oteltrace.Tracer, matchCore *MatchCoreClient, metrics *Metrics) *Server {
	server := &Server{
		config:   config,
		logger:   logger,
		tracer:   tracer,
		matchCore: matchCore,
		metrics:  metrics,
		io:       socketio.New(),
	}
	server.emitToRoom = func(roomCode string, eventName string, args ...any) {
		server.io.To(roomCode).Emit(eventName, args...)
	}

	server.registerHandlers()
	return server
}

func (s *Server) Handler() http.Handler {
	return s.io.HttpHandler()
}

func (s *Server) Close() {
	s.io.Close()
}

func (s *Server) StartTicker(ctx context.Context) context.CancelFunc {
	tickCtx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-tickCtx.Done():
				return
			case <-ticker.C:
				updates, err := s.matchCore.TickActiveRooms(tickCtx)
				if err != nil {
					s.logger.Error("room tick failed", "error", err.Error())
					continue
				}

				for _, snapshot := range updates {
					s.broadcastSnapshot(snapshot.RoomCode, &snapshot)
				}
			}
		}
	}()

	return cancel
}

func (s *Server) registerHandlers() {
	s.io.OnConnection(func(socket *socketio.Socket) {
		if s.metrics != nil {
			s.metrics.IncSocketConnections()
		}

		socket.On("create_room", func(event *socketio.EventPayload) {
			payload, err := decodePayload[CreateRoomPayload](event)
			if err != nil {
				s.sendError("create_room", socket, event, err)
				return
			}
			payload.GameType = normalizeGameType(payload.GameType)

			s.logger.Info(
				"create_room payload decoded",
				"game_type", payload.GameType,
				"nickname", payload.Nickname,
				"mode", payload.Mode,
				"clock_control", payload.ClockControl,
				"device_type", clientFieldValue(payload.Client, "device_type"),
				"platform", clientFieldValue(payload.Client, "platform"),
				"browser", clientFieldValue(payload.Client, "browser"),
				"region", clientFieldValue(payload.Client, "region"),
			)

			s.withAck("create_room", socket, event, func(ctx context.Context) (*operationResult, error) {
				profile := normalizeClientTelemetry(payload.Client)
				snapshot, session, err := s.matchCore.CreateRoom(ctx, payload)
				if err != nil {
					return nil, err
				}
				if s.metrics != nil {
					s.metrics.ObserveRoomCreated(session.Mode, payload.ClockControl, payload.Client)
				}
				s.logger.Info(
					"room_created",
					"game_type", session.GameType,
					"room_code", session.RoomCode,
					"game_id", session.GameID,
					"mode", session.Mode,
					"clock_control", normalizeClockControl(payload.ClockControl),
					"device_type", profile.DeviceType,
					"platform", profile.Platform,
					"browser", profile.Browser,
					"region", profile.Region,
				)

				return &operationResult{
					snapshot: snapshot,
					roomCode: session.RoomCode,
					session:  session,
				}, nil
			})
		})

		socket.On("join_room", func(event *socketio.EventPayload) {
			payload, err := decodePayload[JoinRoomPayload](event)
			if err != nil {
				s.sendError("join_room", socket, event, err)
				return
			}
			payload.GameType = normalizeGameType(payload.GameType)

			s.withAck("join_room", socket, event, func(ctx context.Context) (*operationResult, error) {
				profile := normalizeClientTelemetry(payload.Client)
				snapshot, session, err := s.matchCore.JoinRoom(ctx, payload)
				if err != nil {
					return nil, err
				}
				if s.metrics != nil {
					s.metrics.ObserveRoomJoin(session.Role, session.Mode, payload.Client)
				}
				s.logger.Info(
					"room_joined",
					"game_type", session.GameType,
					"room_code", session.RoomCode,
					"game_id", session.GameID,
					"mode", session.Mode,
					"role", session.Role,
					"device_type", profile.DeviceType,
					"platform", profile.Platform,
					"browser", profile.Browser,
					"region", profile.Region,
				)

				return &operationResult{
					snapshot: snapshot,
					roomCode: session.RoomCode,
					session:  session,
				}, nil
			})
		})

		socket.On("leave_room", func(event *socketio.EventPayload) {
			payload, err := decodePayload[LeaveRoomPayload](event)
			if err != nil {
				s.sendError("leave_room", socket, event, err)
				return
			}

			s.withAck("leave_room", socket, event, func(ctx context.Context) (*operationResult, error) {
				roomCode := normalizeRoomCode(payload.RoomCode)
				snapshot, err := s.matchCore.LeaveRoom(ctx, payload)
				if err != nil {
					return nil, err
				}

				if roomCode != "" {
					socket.Leave(roomCode)
				}
				s.clearSession(socket)
				if snapshot == nil {
					s.clearRoomChatState(roomCode)
				}

				return &operationResult{
					snapshot: snapshot,
					roomCode: roomCode,
					extras: map[string]any{
						"left": true,
					},
				}, nil
			})
		})

		socket.On("submit_action", func(event *socketio.EventPayload) {
			payload, err := decodePayload[SubmitActionPayload](event)
			if err != nil {
				s.sendError("submit_action", socket, event, err)
				return
			}
			payload.GameType = normalizeGameType(payload.GameType)

			s.withAck("submit_action", socket, event, func(ctx context.Context) (*operationResult, error) {
				snapshot, err := s.matchCore.SubmitAction(ctx, payload)
				if err != nil {
					return nil, err
				}

				return &operationResult{
					snapshot: snapshot,
					roomCode: normalizeRoomCode(payload.RoomCode),
				}, nil
			})
		})

		socket.On("resign", func(event *socketio.EventPayload) {
			payload, err := decodePayload[SessionPayload](event)
			if err != nil {
				s.sendError("resign", socket, event, err)
				return
			}

			s.withAck("resign", socket, event, func(ctx context.Context) (*operationResult, error) {
				snapshot, err := s.matchCore.Resign(ctx, payload)
				if err != nil {
					return nil, err
				}

				return &operationResult{snapshot: snapshot, roomCode: normalizeRoomCode(payload.RoomCode)}, nil
			})
		})

		socket.On("offer_draw", func(event *socketio.EventPayload) {
			payload, err := decodePayload[SessionPayload](event)
			if err != nil {
				s.sendError("offer_draw", socket, event, err)
				return
			}

			s.withAck("offer_draw", socket, event, func(ctx context.Context) (*operationResult, error) {
				snapshot, err := s.matchCore.OfferDraw(ctx, payload)
				if err != nil {
					return nil, err
				}

				return &operationResult{snapshot: snapshot, roomCode: normalizeRoomCode(payload.RoomCode)}, nil
			})
		})

		socket.On("accept_draw", func(event *socketio.EventPayload) {
			payload, err := decodePayload[SessionPayload](event)
			if err != nil {
				s.sendError("accept_draw", socket, event, err)
				return
			}

			s.withAck("accept_draw", socket, event, func(ctx context.Context) (*operationResult, error) {
				snapshot, err := s.matchCore.AcceptDraw(ctx, payload)
				if err != nil {
					return nil, err
				}

				return &operationResult{snapshot: snapshot, roomCode: normalizeRoomCode(payload.RoomCode)}, nil
			})
		})

		socket.On("decline_draw", func(event *socketio.EventPayload) {
			payload, err := decodePayload[SessionPayload](event)
			if err != nil {
				s.sendError("decline_draw", socket, event, err)
				return
			}

			s.withAck("decline_draw", socket, event, func(ctx context.Context) (*operationResult, error) {
				snapshot, err := s.matchCore.DeclineDraw(ctx, payload)
				if err != nil {
					return nil, err
				}

				return &operationResult{snapshot: snapshot, roomCode: normalizeRoomCode(payload.RoomCode)}, nil
			})
		})

		socket.On("sync_state", func(event *socketio.EventPayload) {
			payload, err := decodePayload[SessionPayload](event)
			if err != nil {
				s.sendError("sync_state", socket, event, err)
				return
			}

			s.withAck("sync_state", socket, event, func(ctx context.Context) (*operationResult, error) {
				snapshot, session, err := s.matchCore.SyncState(ctx, payload)
				if err != nil {
					return nil, err
				}

				if session != nil {
					return &operationResult{
						snapshot: snapshot,
						roomCode: session.RoomCode,
						session:  session,
					}, nil
				}

				return &operationResult{
					snapshot: snapshot,
					roomCode: normalizeRoomCode(payload.RoomCode),
				}, nil
			})
		})

		socket.On("heartbeat", func(event *socketio.EventPayload) {
			payload, err := decodePayload[HeartbeatPayload](event)
			if err != nil {
				s.sendError("heartbeat", socket, event, err)
				return
			}

			ctx, span := s.startEventSpan(context.Background(), "heartbeat", payload.RoomCode)
			defer span.End()

			snapshot, session, err := s.matchCore.SyncState(ctx, payload)
			if err != nil {
				s.sendError("heartbeat", socket, event, err)
				return
			}

			if session != nil {
				s.bindSession(socket, session)
			}
			if snapshot != nil {
				s.emitChatHistory(socket, normalizeRoomCode(payload.RoomCode), snapshot)
			}

			if event.Ack != nil {
				event.Ack(ackResponse{
					"ok":        true,
					"snapshot":  snapshot,
					"session":   session,
					"serverNow": time.Now().UnixMilli(),
				})
			}
			if s.metrics != nil {
				s.metrics.ObserveSocketEvent("heartbeat", "ok")
			}
		})

		socket.On("chat_message", func(event *socketio.EventPayload) {
			payload, err := decodePayload[ChatMessagePayload](event)
			if err != nil {
				s.sendError("chat_message", socket, event, err)
				return
			}

			s.withAck("chat_message", socket, event, func(ctx context.Context) (*operationResult, error) {
				roomCode, sender, snapshot, err := s.resolveSocketSession(ctx, socket, normalizeGameType(payload.GameType), payload.RoomCode, payload.PlayerToken, payload.SpectatorToken)
				if err != nil {
					return nil, err
				}

				text := strings.TrimSpace(payload.Text)
				if text == "" {
					return nil, &AppError{Message: "Digite uma mensagem antes de enviar.", Code: "EMPTY_CHAT_MESSAGE", StatusCode: 400}
				}
				if len(text) > 500 {
					text = text[:500]
				}

				s.seedChatHistory(roomCode, snapshot)

				message := ChatMessage{
					ID:          firstNonEmpty(payload.MessageID, newChatMessageID()),
					RoomCode:    roomCode,
					SenderName:  firstNonEmpty(strings.TrimSpace(sender.Nickname), "Jogador"),
					SenderType:  senderTypeForRole(sender.Role),
					SenderColor: cloneColor(sender.Color),
					Text:        text,
					Transport:   ChatTransportServer,
					CreatedAt:   time.Now().UnixMilli(),
				}
				if payload.MirrorOnly {
					message.Transport = ChatTransportWebRTC
				}

				stored := s.appendChatMessage(roomCode, message)
				if !payload.MirrorOnly {
					s.emitToRoom(roomCode, "chat_server_message", stored)
				}

				if snapshot != nil && snapshot.Mode == "bot_easy" {
					replyText := buildBotReply(snapshot, text)
					reply := ChatMessage{
						ID:         newChatMessageID(),
						RoomCode:   roomCode,
						SenderName: "Maquina easy",
						SenderType: ChatSenderBot,
						Text:       replyText,
						Transport:  ChatTransportServer,
						CreatedAt:  time.Now().UnixMilli(),
					}
					storedReply := s.appendChatMessage(roomCode, reply)
					s.emitToRoom(roomCode, "chat_server_message", storedReply)
				}

				return &operationResult{
					roomCode: roomCode,
					extras: map[string]any{
						"messageId": stored.ID,
					},
				}, nil
			})
		})

		socket.On("webrtc_signal", func(event *socketio.EventPayload) {
			payload, err := decodePayload[WebRTCSignalPayload](event)
			if err != nil {
				s.sendError("webrtc_signal", socket, event, err)
				return
			}

			ctx, span := s.startEventSpan(context.Background(), "webrtc_signal", payload.RoomCode)
			defer span.End()

			roomCode, sender, _, err := s.resolveSocketSession(ctx, socket, normalizeGameType(payload.GameType), payload.RoomCode, "", "")
			if err != nil {
				s.sendError("webrtc_signal", socket, event, err)
				return
			}

			signal := WebRTCSignalPayload{
				RoomCode:    roomCode,
				Kind:        payload.Kind,
				Description: payload.Description,
				Candidate:   payload.Candidate,
				SenderID:    sender.ClientID,
				SenderColor: cloneColor(sender.Color),
			}
			s.emitToRoom(roomCode, "webrtc_signal", signal)

			if event.Ack != nil {
				event.Ack(ackResponse{"ok": true})
			}
			if s.metrics != nil {
				s.metrics.ObserveSocketEvent("webrtc_signal", "ok")
			}
		})

		socket.On("disconnect", func(event *socketio.EventPayload) {
			if s.metrics != nil {
				s.metrics.DecSocketConnections()
			}
			session, ok := s.loadSession(socket)
			if !ok || session.RoomCode == "" {
				if s.metrics != nil {
					s.metrics.ObserveSocketEvent("disconnect", "ok")
				}
				return
			}

			ctx, span := s.startEventSpan(context.Background(), "disconnect", session.RoomCode)
			defer span.End()

			snapshot, err := s.matchCore.MarkDisconnected(ctx, SessionPayload{
				GameType:       session.GameType,
				RoomCode:       session.RoomCode,
				PlayerToken:    session.PlayerToken,
				SpectatorToken: session.SpectatorToken,
			})
			if err != nil {
				s.logger.Error("disconnect handler failed", "error", err.Error(), "room_code", session.RoomCode)
				return
			}

			if snapshot != nil {
				s.broadcastSnapshot(session.RoomCode, snapshot)
			}
			s.clearSession(socket)
			if s.metrics != nil {
				s.metrics.ObserveSocketEvent("disconnect", "ok")
			}
		})
	})
}

func (s *Server) withAck(
	eventName string,
	socket *socketio.Socket,
	event *socketio.EventPayload,
	operation func(context.Context) (*operationResult, error),
) {
	ctx, span := s.startEventSpan(context.Background(), eventName, "")
	defer span.End()

	result, err := operation(ctx)
	if err != nil {
		s.sendError(eventName, socket, event, err)
		return
	}

	if result != nil && result.session != nil {
		s.bindSession(socket, result.session)
	}

	if result != nil && result.roomCode != "" {
		socket.Join(result.roomCode)
	}

	if result != nil && result.roomCode != "" {
		s.emitChatHistory(socket, result.roomCode, result.snapshot)
	}

	if result != nil && result.snapshot != nil && result.roomCode != "" {
		s.broadcastSnapshot(result.roomCode, result.snapshot)
	}

	if event.Ack != nil {
		response := ackResponse{
			"ok":       true,
			"snapshot": nil,
			"session":  nil,
		}
		if result != nil {
			if result.snapshot != nil {
				response["snapshot"] = result.snapshot
			}
			if result.session != nil {
				response["session"] = result.session
			}
			for key, value := range result.extras {
				response[key] = value
			}
		}
		event.Ack(response)
	}

	if s.metrics != nil {
		s.metrics.ObserveSocketEvent(eventName, "ok")
	}
	s.logger.Info("socket event handled", "event", eventName, "room_code", resultRoomCode(result), "has_snapshot", result != nil && result.snapshot != nil)
}

func (s *Server) sendError(eventName string, socket *socketio.Socket, event *socketio.EventPayload, err error) {
	appErr := &AppError{
		Message:    "Erro interno ao processar a operacao.",
		Code:       "INTERNAL_ERROR",
		StatusCode: 500,
	}
	if errors.As(err, &appErr) {
	} else if concrete, ok := err.(*AppError); ok {
		appErr = concrete
	}

	s.logger.Error("socket event failed", "event", eventName, "code", appErr.Code, "message", appErr.Message, "error", err.Error())
	if s.metrics != nil {
		s.metrics.ObserveSocketEvent(eventName, "error")
	}

	if event.Ack != nil {
		event.Ack(ackResponse{
			"ok":      false,
			"code":    appErr.Code,
			"message": appErr.Message,
		})
	}
}

func (s *Server) broadcastSnapshot(roomCode string, snapshot *RoomSnapshot) {
	if roomCode == "" || snapshot == nil {
		return
	}

	s.seedChatHistory(roomCode, snapshot)
	if botMessage, ok := s.maybeCreateBotMoveComment(roomCode, snapshot); ok {
		s.emitToRoom(roomCode, "chat_server_message", botMessage)
	}
	s.emitToRoom(roomCode, "state_updated", snapshot)
}

func (s *Server) bindSession(socket *socketio.Socket, session *SessionDescriptor) {
	if session == nil {
		return
	}

	normalizedRoomCode := normalizeRoomCode(session.RoomCode)
	previous, hasPrevious := s.loadSession(socket)
	if hasPrevious && previous.RoomCode != "" && previous.RoomCode != normalizedRoomCode {
		socket.Leave(previous.RoomCode)
	}

	clientID := previous.ClientID
	if clientID == "" {
		clientID = newClientID()
	}

	s.sessions.Store(socket, socketSession{
		ClientID:       clientID,
		RoomCode:       normalizedRoomCode,
		PlayerToken:    session.PlayerToken,
		SpectatorToken: session.SpectatorToken,
		Nickname:       session.Nickname,
		Role:           session.Role,
		Color:          cloneColor(session.Color),
		GameType:       session.GameType,
		Mode:           session.Mode,
	})
}

func (s *Server) loadSession(socket *socketio.Socket) (socketSession, bool) {
	value, ok := s.sessions.Load(socket)
	if !ok {
		return socketSession{}, false
	}

	session, ok := value.(socketSession)
	return session, ok
}

func (s *Server) clearSession(socket *socketio.Socket) {
	s.sessions.Delete(socket)
}

func (s *Server) startEventSpan(ctx context.Context, eventName string, roomCode string) (context.Context, oteltrace.Span) {
	ctx, span := s.tracer.Start(ctx, "socket."+eventName)
	if roomCode != "" {
		span.SetAttributes(attribute.String("room.code", roomCode))
	}
	return ctx, span
}

func (s *Server) resolveSocketSession(
	ctx context.Context,
	socket *socketio.Socket,
	gameType GameType,
	roomCode string,
	playerToken string,
	spectatorToken string,
) (string, socketSession, *RoomSnapshot, error) {
	normalizedRoomCode := normalizeRoomCode(roomCode)
	if current, ok := s.loadSession(socket); ok {
		if normalizedRoomCode == "" || current.RoomCode == normalizedRoomCode {
			snapshot, session, err := s.matchCore.SyncState(ctx, SessionPayload{
				GameType:       current.GameType,
				RoomCode:       current.RoomCode,
				PlayerToken:    current.PlayerToken,
				SpectatorToken: current.SpectatorToken,
			})
			if err != nil {
				return "", socketSession{}, nil, err
			}
			if session != nil {
				s.bindSession(socket, session)
				current, _ = s.loadSession(socket)
			}
			return current.RoomCode, current, snapshot, nil
		}
	}

	if normalizedRoomCode == "" {
		return "", socketSession{}, nil, &AppError{Message: "Sala nao informada.", Code: "BAD_REQUEST", StatusCode: 400}
	}

	snapshot, session, err := s.matchCore.SyncState(ctx, SessionPayload{
		GameType:       normalizeGameType(gameType),
		RoomCode:       normalizedRoomCode,
		PlayerToken:    playerToken,
		SpectatorToken: spectatorToken,
	})
	if err != nil {
		return "", socketSession{}, nil, err
	}
	if session == nil {
		return "", socketSession{}, nil, &AppError{Message: "Sessao invalida para a sala.", Code: "SESSION_REQUIRED", StatusCode: 403}
	}

	s.bindSession(socket, session)
	socket.Join(session.RoomCode)
	bound, _ := s.loadSession(socket)
	return bound.RoomCode, bound, snapshot, nil
}

func (s *Server) roomChat(roomCode string) *roomChatState {
	normalizedRoomCode := normalizeRoomCode(roomCode)
	value, _ := s.chatRooms.LoadOrStore(normalizedRoomCode, &roomChatState{
		history: make([]ChatMessage, 0, 8),
	})

	state, _ := value.(*roomChatState)
	return state
}

func (s *Server) clearRoomChatState(roomCode string) {
	if roomCode == "" {
		return
	}

	s.chatRooms.Delete(normalizeRoomCode(roomCode))
}

func (s *Server) emitChatHistory(socket *socketio.Socket, roomCode string, snapshot *RoomSnapshot) {
	if socket == nil || roomCode == "" {
		return
	}

	history := s.seedChatHistory(roomCode, snapshot)
	socket.Emit("chat_history", history)
}

func (s *Server) seedChatHistory(roomCode string, snapshot *RoomSnapshot) []ChatMessage {
	state := s.roomChat(roomCode)
	state.mu.Lock()
	defer state.mu.Unlock()

	if !state.seeded {
		now := time.Now().UnixMilli()
		state.history = append(state.history, ChatMessage{
			ID:         newChatMessageID(),
			RoomCode:   roomCode,
			SenderName: "Sistema",
			SenderType: ChatSenderSystem,
			Text:       "Chat ativo nesta sala. Em partidas PvP tentamos abrir um canal WebRTC entre os jogadores; se nao houver conexao direta, o chat continua pelo servidor.",
			Transport:  ChatTransportServer,
			CreatedAt:  now,
		})
		if snapshot != nil && snapshot.Mode == "bot_easy" {
			state.history = append(state.history, ChatMessage{
				ID:         newChatMessageID(),
				RoomCode:   roomCode,
				SenderName: "Maquina easy",
				SenderType: ChatSenderBot,
				Text:       buildBotWelcomeMessage(snapshot),
				Transport:  ChatTransportServer,
				CreatedAt:  now + 1,
			})
		}
		state.seeded = true
	}

	return cloneChatMessages(state.history)
}

func (s *Server) appendChatMessage(roomCode string, message ChatMessage) ChatMessage {
	state := s.roomChat(roomCode)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.history = append(state.history, message)
	if len(state.history) > maxChatHistory {
		state.history = append([]ChatMessage(nil), state.history[len(state.history)-maxChatHistory:]...)
	}

	return message
}

func (s *Server) maybeCreateBotMoveComment(roomCode string, snapshot *RoomSnapshot) (ChatMessage, bool) {
	if snapshot == nil || snapshot.Mode != "bot_easy" || snapshot.LastMove == nil {
		return ChatMessage{}, false
	}

	state := s.roomChat(roomCode)
	state.mu.Lock()
	defer state.mu.Unlock()

	if snapshot.LastMove.At == 0 || state.lastBotCommentAt == snapshot.LastMove.At {
		return ChatMessage{}, false
	}

	message := ChatMessage{
		ID:         newChatMessageID(),
		RoomCode:   roomCode,
		SenderName: "Maquina easy",
		SenderType: ChatSenderBot,
		Text:       buildBotMoveComment(snapshot),
		Transport:  ChatTransportServer,
		CreatedAt:  time.Now().UnixMilli(),
	}
	state.history = append(state.history, message)
	if len(state.history) > maxChatHistory {
		state.history = append([]ChatMessage(nil), state.history[len(state.history)-maxChatHistory:]...)
	}
	state.lastBotCommentAt = snapshot.LastMove.At

	return message, true
}

func decodePayload[T any](event *socketio.EventPayload) (T, error) {
	var payload T
	if event == nil || len(event.Data) == 0 {
		return payload, &AppError{Message: "Payload ausente.", Code: "BAD_REQUEST", StatusCode: 400}
	}

	encoded, err := json.Marshal(event.Data[0])
	if err != nil {
		return payload, err
	}

	if err := json.Unmarshal(encoded, &payload); err != nil {
		return payload, &AppError{Message: "Payload invalido.", Code: "BAD_REQUEST", StatusCode: 400}
	}

	return payload, nil
}

func resultRoomCode(result *operationResult) string {
	if result == nil {
		return ""
	}

	return result.roomCode
}

func clientFieldValue(client *ClientTelemetry, field string) string {
	if client == nil {
		return "unknown"
	}

	switch field {
	case "device_type":
		return client.DeviceType
	case "platform":
		return client.Platform
	case "browser":
		return client.Browser
	case "region":
		return client.Region
	default:
		return "unknown"
	}
}

func normalizeRoomCode(roomCode string) string {
	return strings.ToUpper(strings.TrimSpace(roomCode))
}

func normalizeGameType(gameType GameType) GameType {
	normalized := strings.TrimSpace(string(gameType))
	if normalized == "" {
		return GameType("chess")
	}
	return GameType(strings.ToLower(normalized))
}

func cloneColor(color *Color) *Color {
	if color == nil {
		return nil
	}

	cloned := *color
	return &cloned
}

func cloneChatMessages(messages []ChatMessage) []ChatMessage {
	cloned := make([]ChatMessage, len(messages))
	copy(cloned, messages)
	return cloned
}

func newClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}

func newChatMessageID() string {
	return fmt.Sprintf("chat-%d", time.Now().UnixNano())
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

func senderTypeForRole(role ViewerRole) ChatSenderType {
	if role == "spectator" {
		return ChatSenderSpectator
	}

	return ChatSenderPlayer
}

func buildBotReply(snapshot *RoomSnapshot, text string) string {
	if snapshot != nil && snapshot.GameType == "tictactoe" {
		return buildTicTacToeBotReply(snapshot, text)
	}

	return buildChessBotReply(snapshot, text)
}

func buildChessBotReply(snapshot *RoomSnapshot, text string) string {
	lower := strings.ToLower(text)

	switch {
	case strings.Contains(lower, "roque"):
		return "Roque move o rei duas casas em direcao a torre e a torre salta para o outro lado. Ele so e permitido se rei e torre ainda nao se moveram, o caminho estiver livre e o rei nao passar por xeque."
	case strings.Contains(lower, "xeque"):
		return "Xeque significa que o rei esta atacado. A resposta precisa tirar o rei do ataque: mover o rei, capturar a peca atacante ou bloquear a linha de ataque."
	case strings.Contains(lower, "empate"):
		return "No xadrez voce pode empatar por acordo, afogamento, repeticao, regra dos 50 lances ou material insuficiente."
	case strings.Contains(lower, "rei"):
		return "O rei anda uma casa por vez em qualquer direcao. Ele nunca pode terminar o lance em uma casa atacada."
	case strings.Contains(lower, "dama"):
		return "A dama combina o movimento da torre e do bispo. Ela e forte, mas expor a dama cedo demais costuma custar tempo."
	case strings.Contains(lower, "torre"):
		return "A torre anda em linhas retas. Ela cresce muito depois que os peoes abrem colunas."
	case strings.Contains(lower, "bispo"):
		return "O bispo anda nas diagonais. Bispos gostam de diagonais longas e de peoes que nao bloqueiem seu caminho."
	case strings.Contains(lower, "cavalo"):
		return "O cavalo anda em L e pode saltar sobre outras pecas. No inicio, ele ajuda muito a disputar casas centrais."
	case strings.Contains(lower, "peao"), strings.Contains(lower, "peao"), strings.Contains(lower, "abertura"), strings.Contains(lower, "inicio"):
		return "Para iniciantes, a regra pratica e simples: ocupe o centro com peoes, desenvolva cavalos e bispos e tente deixar o rei seguro cedo."
	case snapshot != nil && snapshot.LastMove != nil && snapshot.LastMove.Color == "w":
		return fmt.Sprintf("Seu ultimo lance foi %s. Pergunte sobre ele se quiser: posso explicar a ideia taticamente e tambem lembrar a regra envolvida.", snapshot.LastMove.SAN)
	case snapshot != nil && snapshot.LastMove != nil:
		return fmt.Sprintf("A ultima jogada da sala foi %s. Observe quais casas ela passou a controlar e se abriu espaco para desenvolver outra peca.", snapshot.LastMove.SAN)
	default:
		return "Posso explicar regras basicas e comentar o ultimo lance. Pergunte, por exemplo: 'como funciona roque?', 'o que e xeque?' ou 'o que voce achou do meu ultimo lance?'."
	}
}

func buildBotMoveComment(snapshot *RoomSnapshot) string {
	if snapshot != nil && snapshot.GameType == "tictactoe" {
		return buildTicTacToeBotMoveComment(snapshot)
	}

	return buildChessBotMoveComment(snapshot)
}

func buildChessBotMoveComment(snapshot *RoomSnapshot) string {
	if snapshot == nil || snapshot.LastMove == nil {
		return "Siga desenvolvendo suas pecas e cuide do centro."
	}

	move := snapshot.LastMove
	if move.Color == "w" {
		return fmt.Sprintf("Seu lance %s foi registrado. Em geral, tente desenvolver pecas leves e manter o rei seguro cedo.", move.SAN)
	}

	return fmt.Sprintf("Respondi com %s. Repare como um unico lance pode ganhar espaco, defender uma peca ou preparar o desenvolvimento.", move.SAN)
}

func buildBotWelcomeMessage(snapshot *RoomSnapshot) string {
	if snapshot != nil && snapshot.GameType == "tictactoe" {
		return "Posso comentar suas jogadas no jogo da velha, lembrar as regras e sugerir bloqueios, linhas de vitoria e disputas pelo centro."
	}

	return "Posso comentar seus lances e explicar regras basicas. Pergunte sobre aberturas, xeque, roque, empate ou sobre a ultima jogada."
}

func buildTicTacToeBotReply(snapshot *RoomSnapshot, text string) string {
	lower := strings.ToLower(text)

	switch {
	case strings.Contains(lower, "regra"), strings.Contains(lower, "como joga"):
		return "No jogo da velha, vence quem completar tres marcas em linha, coluna ou diagonal. Se o tabuleiro encher sem linha completa, termina empatado."
	case strings.Contains(lower, "centro"):
		return "O centro costuma ser a melhor casa inicial porque participa de quatro linhas vencedoras. Se ele estiver livre, quase sempre vale disputa-lo."
	case strings.Contains(lower, "bloque"), strings.Contains(lower, "defes"):
		return "Defender no jogo da velha costuma significar bloquear uma linha com duas marcas do adversario antes do terceiro lance."
	case strings.Contains(lower, "fork"), strings.Contains(lower, "dupla"):
		return "Uma bifurcacao acontece quando um lance cria duas ameacas ao mesmo tempo. Se voce puder criar duas linhas abertas, o rival nao consegue bloquear ambas."
	case snapshot != nil && snapshot.LastMove != nil && snapshot.LastMove.Color == "w":
		return fmt.Sprintf("Seu ultimo lance foi %s. Veja se ele abriu duas linhas ao mesmo tempo ou se deixou uma resposta obrigatoria para o rival.", snapshot.LastMove.SAN)
	case snapshot != nil && snapshot.LastMove != nil:
		return fmt.Sprintf("A ultima jogada da sala foi %s. Confira se ela bloqueou uma linha sua ou se criou uma ameaca imediata.", snapshot.LastMove.SAN)
	default:
		return "Posso comentar a ultima jogada, lembrar as regras e sugerir bloqueios. Pergunte, por exemplo: 'devo disputar o centro?' ou 'como evitar uma bifurcacao?'."
	}
}

func buildTicTacToeBotMoveComment(snapshot *RoomSnapshot) string {
	if snapshot == nil || snapshot.LastMove == nil {
		return "No jogo da velha, disputar centro e cantos cedo costuma simplificar as linhas de vitoria."
	}

	move := snapshot.LastMove
	if move.Color == "w" {
		return fmt.Sprintf("Seu lance %s foi registrado. Agora confira se ele criou uma segunda ameaca ou se ainda precisa bloquear alguma linha aberta.", move.SAN)
	}

	return fmt.Sprintf("Respondi com %s. No jogo da velha, cada casa muda varias linhas ao mesmo tempo, entao vale revisar o tabuleiro inteiro a cada jogada.", move.SAN)
}
