package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Marques-net/geek-hub/services/realtime-gateway/internal/gateway"
	"github.com/Marques-net/geek-hub/services/realtime-gateway/internal/observability"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	cfg := gateway.LoadConfig()
	logger := observability.NewLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := observability.SetupTracing(ctx, logger)
	if err != nil {
		logger.Error("failed to setup tracing", "error", err.Error())
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			logger.Error("failed to shutdown tracing", "error", err.Error())
		}
	}()

	metrics := gateway.NewMetrics()
	matchCore, err := gateway.NewMatchCoreClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if closeErr := matchCore.Close(); closeErr != nil {
			logger.Error("failed to close match core connection", "error", closeErr.Error())
		}
	}()

	if err := matchCore.Ready(ctx); err != nil {
		log.Fatal(err)
	}

	server := gateway.NewServer(cfg, logger, observability.Tracer("games-realtime-gateway"), matchCore, metrics)
	defer server.Close()
	stopTicker := server.StartTicker(ctx)
	defer stopTicker()

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/socket.io/", server.Handler())
	mux.Handle("/socket.io", server.Handler())
	mux.HandleFunc("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": "realtime-gateway",
		})
	})
	mux.HandleFunc("/health/ready", func(writer http.ResponseWriter, request *http.Request) {
		checkCtx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := matchCore.Ready(checkCtx); err != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
				"status":   "error",
				"matchCore": "unavailable",
				"message":  err.Error(),
			})
			return
		}

		writeJSON(writer, http.StatusOK, map[string]any{
			"status":   "ok",
			"matchCore": "ready",
		})
	})
	mux.HandleFunc("/api/info", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{
			"name":          "geek-hub-realtime-gateway",
			"websocketPath": "/socket.io",
			"clockSeconds":  cfg.RoomClockSeconds,
		})
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           withHTTPObservability(logger, observability.Tracer("games-realtime-gateway.http"), metrics, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("http shutdown failed", "error", err.Error())
		}
	}()

	logger.Info("realtime-gateway started", "port", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload map[string]any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func withHTTPObservability(logger *slog.Logger, tracer trace.Tracer, metrics *gateway.Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx, span := tracer.Start(request.Context(), request.Method+" "+request.URL.Path)
		defer span.End()

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: writer, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, request.WithContext(ctx))
		if metrics != nil {
			metrics.ObserveHTTPRequest(request.Method, request.URL.Path, recorder.statusCode, time.Since(start))
		}

		spanContext := span.SpanContext()
		logger.Info(
			"http request completed",
			"method", request.Method,
			"path", request.URL.Path,
			"status_code", recorder.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"trace_id", spanContext.TraceID().String(),
			"span_id", spanContext.SpanID().String(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}

	return hijacker.Hijack()
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
