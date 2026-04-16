package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oteltrace "go.opentelemetry.io/otel/trace"
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

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func SetupTracing(ctx context.Context, logger *slog.Logger) (func(context.Context) error, error) {
	endpoint := envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "tempo.monitoring.svc.cluster.local:4317")
	serviceName := envOr("OTEL_SERVICE_NAME", "geek-hub-login-service")

	options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if strings.EqualFold(envOr("OTEL_EXPORTER_OTLP_INSECURE", "true"), "true") {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	traceCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(traceCtx, options...)
	if err != nil {
		return func(context.Context) error { return nil }, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("0.1.0-go"),
			attribute.String("deployment.environment", envOr("APP_ENV", "production")),
			attribute.String("k8s.namespace.name", envOr("POD_NAMESPACE", "unknown")),
			attribute.String("k8s.pod.name", envOr("POD_NAME", "unknown")),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	logger.Info("tempo tracing enabled", "endpoint", endpoint, "service_name", serviceName)
	return provider.Shutdown, nil
}

func Tracer(name string) oteltrace.Tracer {
	return otel.Tracer(name)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
