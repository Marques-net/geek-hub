package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	chessengine "github.com/Marques-net/geek-hub/services/bot-engine/internal/games/chess"
	"github.com/Marques-net/geek-hub/services/bot-engine/internal/observability"
	strategyv1 "github.com/Marques-net/geek-hub/services/bot-engine/proto/strategyv1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type strategyEngineServer struct {
	strategyv1.UnimplementedStrategyEngineServiceServer
	logger *slog.Logger
	tracer trace.Tracer
}

func (s *strategyEngineServer) GetAction(
	ctx context.Context,
	req *strategyv1.GetActionRequest,
) (*strategyv1.GetActionResponse, error) {
	ctx, span := s.tracer.Start(
		ctx,
		"strategy_engine.get_action",
		trace.WithAttributes(
			attribute.String("games.game_type", req.GetGameType()),
			attribute.String("game.id", req.GetGameId()),
			attribute.String("chess.room_code", req.GetRoomCode()),
			attribute.String("room.code", req.GetRoomCode()),
			attribute.String("engine.mode", req.GetMode()),
			attribute.Int64("move.count", int64(req.GetActionCount())),
		),
	)
	defer span.End()

	if req.GetGameType() != "chess" {
		err := status.Errorf(grpcCodes.InvalidArgument, "unsupported game type: %s", req.GetGameType())
		span.RecordError(err)
		span.SetStatus(codes.Error, "unsupported game type")
		return nil, err
	}
	if req.GetMode() != "bot_easy" {
		err := status.Errorf(grpcCodes.InvalidArgument, "unsupported strategy mode: %s", req.GetMode())
		span.RecordError(err)
		span.SetStatus(codes.Error, "unsupported mode")
		return nil, err
	}

	var statePayload struct {
		FEN string `json:"fen"`
	}
	if err := json.Unmarshal([]byte(req.GetStateJson()), &statePayload); err != nil {
		err = status.Errorf(grpcCodes.InvalidArgument, "invalid state payload: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid payload")
		return nil, err
	}

	move, err := chessengine.SelectEasyMove(statePayload.FEN)
	if err != nil {
		err = status.Errorf(grpcCodes.InvalidArgument, "invalid request: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request")
		return nil, err
	}

	if move == nil {
		span.SetAttributes(attribute.Bool("move.found", false))
		return &strategyv1.GetActionResponse{
			Found: false,
		}, nil
	}

	actionPayload, err := json.Marshal(map[string]string{
		"from":      move.From,
		"to":        move.To,
		"promotion": move.Promotion,
	})
	if err != nil {
		return nil, err
	}

	span.SetAttributes(
		attribute.Bool("move.found", true),
		attribute.String("move.from", move.From),
		attribute.String("move.to", move.To),
		attribute.String("move.promotion", move.Promotion),
	)

	return &strategyv1.GetActionResponse{
		Found:             true,
		ActionType:        "move",
		ActionPayloadJson: string(actionPayload),
		CoachMessage:      "Priorize desenvolver pecas leves, proteger o rei e disputar o centro.",
	}, nil
}

func main() {
	port := envOr("PORT", "50051")
	logger := observability.NewLogger(envOr("LOG_LEVEL", "INFO"))

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

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to bind listener", "port", port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(observability.NewGRPCStatsHandler()),
		grpc.ChainUnaryInterceptor(observability.LoggingUnaryInterceptor(logger)),
	)

	strategyv1.RegisterStrategyEngineServiceServer(grpcServer, &strategyEngineServer{
		logger: logger,
		tracer: observability.Tracer("bot-engine"),
	})

	logger.Info(
		"bot engine started",
		"port", port,
		"tempo_endpoint", envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "tempo.monitoring.svc.cluster.local:4317"),
		"log_pipeline", "stdout->promtail->loki",
		"trace_pipeline", "otlp->tempo",
	)

	serverErrors := make(chan error, 1)
	go func() {
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
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

	select {
	case <-done:
		logger.Info("bot engine stopped gracefully")
	case <-time.After(10 * time.Second):
		logger.Warn("forcing grpc server stop after timeout")
		grpcServer.Stop()
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
