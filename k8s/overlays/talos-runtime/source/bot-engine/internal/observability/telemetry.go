package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otelTrace "go.opentelemetry.io/otel/trace"
	strategyv1 "github.com/Marques-net/geek-hub/services/bot-engine/proto/strategyv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
)

func NewLogger(levelValue string) *slog.Logger {
	level := slog.LevelInfo

	switch strings.ToUpper(levelValue) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

func SetupTracing(ctx context.Context, logger *slog.Logger) (func(context.Context) error, error) {
	endpoint := envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "tempo.monitoring.svc.cluster.local:4317")
	serviceName := envOr("OTEL_SERVICE_NAME", "games-bot-engine")

	exporterOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}

	if strings.EqualFold(envOr("OTEL_EXPORTER_OTLP_INSECURE", "true"), "true") {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}

	traceCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(traceCtx, exporterOptions...)
	if err != nil {
		return func(context.Context) error { return nil }, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("0.3.0-go"),
			attribute.String("deployment.environment", envOr("APP_ENV", "production")),
			attribute.String("k8s.namespace.name", envOr("POD_NAMESPACE", "unknown")),
			attribute.String("k8s.pod.name", envOr("POD_NAME", "unknown")),
		),
	)
	if err != nil {
		logger.Warn("failed to merge tracing resources", "error", err)
		res = resource.Default()
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	logger.Info("tempo tracing enabled", "endpoint", endpoint, "service_name", serviceName)

	return provider.Shutdown, nil
}

func NewGRPCStatsHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

func LoggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		spanContext := otelTrace.SpanContextFromContext(ctx)
		fields := []any{
			"grpc.method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"trace_id", spanContext.TraceID().String(),
			"span_id", spanContext.SpanID().String(),
		}

		if peerInfo, ok := peer.FromContext(ctx); ok && peerInfo.Addr != nil {
			fields = append(fields, "peer.address", peerInfo.Addr.String())
		}

		if request, ok := req.(*strategyv1.GetActionRequest); ok {
			fields = append(fields,
				"room_code", request.GetRoomCode(),
				"game_id", request.GetGameId(),
				"mode", request.GetMode(),
				"move_count", request.GetActionCount(),
			)
		}

		if err != nil {
			logger.Error("grpc request failed", append(fields, "error", err.Error())...)
			return resp, err
		}

		logger.Info("grpc request completed", fields...)
		return resp, nil
	}
}

func Tracer(name string) otelTrace.Tracer {
	return otel.Tracer(name)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
