package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Marques-net/geek-hub/services/login-service/internal/login"
	"github.com/Marques-net/geek-hub/services/login-service/internal/observability"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	cfg := login.LoadConfig()
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

	metrics := login.NewMetrics()
	loginRepository, err := login.NewLoginRepository(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := loginRepository.Close(shutdownCtx); err != nil {
			logger.Error("failed to close login repository", "error", err.Error())
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": "login-service",
		})
	})
	mux.HandleFunc("/health/ready", func(writer http.ResponseWriter, request *http.Request) {
		checkCtx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := loginRepository.Ready(checkCtx); err != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
				"status":  "error",
				"mongo":   "unavailable",
				"message": err.Error(),
			})
			return
		}

		writeJSON(writer, http.StatusOK, map[string]any{
			"status": "ok",
			"mongo":  "ready",
		})
	})
	mux.HandleFunc("/api/info", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{
			"name": "geek-hub-login-service",
		})
	})
	mux.HandleFunc("/api/auth/logins", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]any{
				"status":  "error",
				"message": "Metodo nao suportado.",
			})
			return
		}

		payload, err := decodeLoginRequest(request.Body)
		if err != nil {
			metrics.ObserveLoginRecord(payloadProvider(payload), "invalid")
			writeJSON(writer, http.StatusBadRequest, map[string]any{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}

		loginCtx, cancel := context.WithTimeout(request.Context(), 3*time.Second)
		defer cancel()

		loggedAt := time.Now().UTC()
		rawUserAgent := payload.Client.RawUserAgent
		if rawUserAgent == "" || rawUserAgent == "unknown" {
			rawUserAgent = sanitizeText(request.UserAgent(), 1024)
		}

		if err := loginRepository.RecordLogin(loginCtx, login.UserLoginRecord{
			Provider:        payload.Provider,
			ProviderUserID:  payload.Sub,
			Name:            payload.Name,
			Email:           payload.Email,
			LoggedAt:        loggedAt,
			Source:          "portal-web",
			UserAgent:       request.UserAgent(),
			DeviceType:      payload.Client.DeviceType,
			Platform:        payload.Client.Platform,
			PlatformVersion: payload.Client.PlatformVersion,
			Browser:         payload.Client.Browser,
			BrowserVersion:  payload.Client.BrowserVersion,
			Region:          payload.Client.Region,
			DeviceModel:     payload.Client.DeviceModel,
			RawUserAgent:    rawUserAgent,
		}); err != nil {
			metrics.ObserveLoginRecord(payload.Provider, "error")
			logger.Error("failed to persist login event", "error", err.Error(), "provider", payload.Provider)
			writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
				"status":  "error",
				"message": "Nao foi possivel registrar o login do usuario.",
			})
			return
		}

		metrics.ObserveLoginRecord(payload.Provider, "ok")
		writeJSON(writer, http.StatusCreated, map[string]any{
			"status":   "ok",
			"provider": payload.Provider,
			"loggedAt": loggedAt.Format(time.RFC3339),
		})
	})
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           withHTTPObservability(logger, observability.Tracer("login-service.http"), metrics, mux),
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

	logger.Info("login-service started", "port", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload map[string]any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

type loginRequest struct {
	Provider string                 `json:"provider"`
	Sub      string                 `json:"sub"`
	Name     string                 `json:"name"`
	Email    *string                `json:"email"`
	Client   clientTelemetryRequest `json:"client"`
}

type clientTelemetryRequest struct {
	DeviceType      string `json:"deviceType"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	Browser         string `json:"browser"`
	BrowserVersion  string `json:"browserVersion"`
	Region          string `json:"region"`
	DeviceModel     string `json:"deviceModel"`
	RawUserAgent    string `json:"rawUserAgent"`
}

func decodeLoginRequest(body io.ReadCloser) (*loginRequest, error) {
	defer body.Close()

	var payload loginRequest
	decoder := json.NewDecoder(io.LimitReader(body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return nil, errors.New("Payload de login invalido.")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("Payload de login invalido.")
	}

	payload.Provider = strings.TrimSpace(strings.ToLower(payload.Provider))
	payload.Sub = strings.TrimSpace(payload.Sub)
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Email != nil {
		email := strings.TrimSpace(*payload.Email)
		if email == "" {
			payload.Email = nil
		} else {
			payload.Email = &email
		}
	}
	payload.Client.DeviceType = normalizeDimension(payload.Client.DeviceType)
	payload.Client.Platform = normalizeDimension(payload.Client.Platform)
	payload.Client.PlatformVersion = sanitizeText(payload.Client.PlatformVersion, 64)
	payload.Client.Browser = normalizeDimension(payload.Client.Browser)
	payload.Client.BrowserVersion = sanitizeText(payload.Client.BrowserVersion, 64)
	payload.Client.Region = normalizeDimension(payload.Client.Region)
	payload.Client.DeviceModel = sanitizeText(payload.Client.DeviceModel, 160)
	payload.Client.RawUserAgent = sanitizeText(payload.Client.RawUserAgent, 1024)

	switch {
	case payload.Provider != "google":
		return nil, errors.New("Somente login Google pode ser registrado nesta rota.")
	case payload.Sub == "":
		return nil, errors.New("Identificador do usuario Google e obrigatorio.")
	case payload.Name == "":
		return nil, errors.New("Nome do usuario e obrigatorio.")
	default:
		return &payload, nil
	}
}

func payloadProvider(payload *loginRequest) string {
	if payload == nil {
		return "unknown"
	}

	return payload.Provider
}

func normalizeDimension(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "unknown"
	}

	return normalized
}

func sanitizeText(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) > limit {
		return string(runes[:limit])
	}

	return trimmed
}

func withHTTPObservability(logger *slog.Logger, tracer trace.Tracer, metrics *login.Metrics, next http.Handler) http.Handler {
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
