package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Marques-net/geek-hub/services/match-core/internal/games/chess"
	"github.com/Marques-net/geek-hub/services/match-core/internal/observability"
	"github.com/Marques-net/geek-hub/services/match-core/internal/platform"
	matchcorev1 "github.com/Marques-net/geek-hub/services/match-core/proto/matchcore"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type server struct {
	matchcorev1.UnimplementedMatchCoreServiceServer
	logger   *slog.Logger
	tracer   trace.Tracer
	registry *platform.Registry
}

func (s *server) runtimeFor(gameType string) platform.Runtime {
	resolved := gameType
	if resolved == "" {
		resolved = chess.GameTypeChess
	}
	return s.registry.Resolve(resolved)
}

func (s *server) withCommand(ctx context.Context, name string, req *matchcorev1.RoomRequest, handler func(platform.Runtime, context.Context, *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error)) (*matchcorev1.RoomResponse, error) {
	runtime := s.runtimeFor(req.GetGameType())
	if runtime == nil {
		return &matchcorev1.RoomResponse{
			Ok:         false,
			Code:       "UNSUPPORTED_GAME_TYPE",
			Message:    "Tipo de jogo não suportado.",
			StatusCode: 400,
		}, nil
	}

	ctx, span := s.tracer.Start(ctx, name, trace.WithAttributes(
		attribute.String("games.game_type", runtime.GameType()),
		attribute.String("chess.room_code", req.GetRoomCode()),
		attribute.String("room.code", req.GetRoomCode()),
		attribute.String("mode", req.GetMode()),
		attribute.String("clock.control", req.GetClockControl()),
		attribute.String("action.type", req.GetActionType()),
	))
	defer span.End()

	s.logger.Info(
		"match-core command received",
		"name", name,
		"game_type", runtime.GameType(),
		"room_code", req.GetRoomCode(),
		"mode", req.GetMode(),
		"clock_control", req.GetClockControl(),
		"action_type", req.GetActionType(),
	)

	response, err := handler(runtime, ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "handler error")
		return nil, err
	}

	if !response.GetOk() {
		span.SetStatus(codes.Error, response.GetCode())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return response, nil
}

func (s *server) Ready(ctx context.Context, _ *matchcorev1.TickRequest) (*matchcorev1.RoomResponse, error) {
	runtime := s.runtimeFor(chess.GameTypeChess)
	if runtime == nil {
		return &matchcorev1.RoomResponse{
			Ok:         false,
			Code:       "RUNTIME_UNAVAILABLE",
			Message:    "Nenhum runtime registrado.",
			StatusCode: 500,
		}, nil
	}
	return runtime.Ready(ctx)
}

func (s *server) CreateRoom(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.create_room", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.CreateRoom(commandCtx, request)
	})
}

func (s *server) JoinRoom(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.join_room", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.JoinRoom(commandCtx, request)
	})
}

func (s *server) LeaveRoom(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.leave_room", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.LeaveRoom(commandCtx, request)
	})
}

func (s *server) SyncState(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.sync_state", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.SyncState(commandCtx, request)
	})
}

func (s *server) SubmitAction(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.submit_action", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.SubmitAction(commandCtx, request)
	})
}

func (s *server) Resign(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.resign", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.Resign(commandCtx, request)
	})
}

func (s *server) OfferDraw(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.offer_draw", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.OfferDraw(commandCtx, request)
	})
}

func (s *server) AcceptDraw(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.accept_draw", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.AcceptDraw(commandCtx, request)
	})
}

func (s *server) DeclineDraw(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.decline_draw", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.DeclineDraw(commandCtx, request)
	})
}

func (s *server) MarkDisconnected(ctx context.Context, req *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
	return s.withCommand(ctx, "match_core.mark_disconnected", req, func(runtime platform.Runtime, commandCtx context.Context, request *matchcorev1.RoomRequest) (*matchcorev1.RoomResponse, error) {
		return runtime.MarkDisconnected(commandCtx, request)
	})
}

func (s *server) TickActiveRooms(ctx context.Context, _ *matchcorev1.TickRequest) (*matchcorev1.TickResponse, error) {
	ctx, span := s.tracer.Start(ctx, "match_core.tick_active_rooms")
	defer span.End()

	runtime := s.runtimeFor(chess.GameTypeChess)
	if runtime == nil {
		span.SetStatus(codes.Error, "runtime unavailable")
		return &matchcorev1.TickResponse{}, nil
	}

	response, err := runtime.TickActiveRooms(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "tick failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return response, nil
}

func main() {
	config := chess.LoadConfig()
	logger := observability.NewLogger(config.LogLevel)

	ctx := context.Background()
	shutdownTracing, err := observability.SetupTracing(ctx, logger)
	if err != nil {
		logger.Error("failed to configure tracing", "error", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracing(shutdownCtx)
	}()

	store := chess.NewStore(config)
	if err := store.Connect(ctx); err != nil {
		logger.Error("failed to connect redis", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	botClient, err := chess.NewBotClient(config)
	if err != nil {
		logger.Error("failed to create bot engine client", "error", err)
		os.Exit(1)
	}
	defer botClient.Close()

	metrics := chess.NewMetrics()
	chessRuntime := chess.NewService(config, store, botClient, metrics)
	if err := chessRuntime.PrimeMetrics(ctx); err != nil {
		logger.Error("failed to prime room metrics", "error", err)
	}

	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		logger.Error("failed to bind listener", "port", config.Port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(observability.NewGRPCStatsHandler()),
		grpc.ChainUnaryInterceptor(observability.LoggingUnaryInterceptor(logger)),
	)

	matchcorev1.RegisterMatchCoreServiceServer(grpcServer, &server{
		logger:   logger,
		tracer:   observability.Tracer("match-core"),
		registry: platform.NewRegistry(chessRuntime),
	})

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})
	metricsServer := &http.Server{
		Addr:              ":" + config.MetricsPort,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("match core started", "port", config.Port, "metrics_port", config.MetricsPort, "log_pipeline", "stdout->promtail->loki", "trace_pipeline", "otlp->tempo")

	serverErrors := make(chan error, 1)
	go func() {
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			serverErrors <- serveErr
		}
	}()
	go func() {
		if serveErr := metricsServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErrors <- serveErr
		}
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	case serveErr := <-serverErrors:
		if !errors.Is(serveErr, grpc.ErrServerStopped) {
			logger.Error("grpc server stopped unexpectedly", "error", serveErr)
			os.Exit(1)
		}
	}

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Warn("metrics server shutdown failed", "error", err)
	}

	select {
	case <-done:
		logger.Info("match core stopped gracefully")
	case <-time.After(10 * time.Second):
		logger.Warn("forcing grpc server stop after timeout")
		grpcServer.Stop()
	}
}
