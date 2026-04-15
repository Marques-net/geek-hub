package gateway

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	socketEventsTotal    *prometheus.CounterVec
	socketConnectionsNow prometheus.Gauge
	roomCreationsTotal   *prometheus.CounterVec
	roomJoinsTotal       *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		httpRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "chess_backend_http_requests_total",
			Help: "HTTP requests handled by the realtime gateway.",
		}, []string{"method", "path", "status_code"}),
		httpRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "chess_backend_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds for the realtime gateway.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		socketEventsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "chess_backend_socket_events_total",
			Help: "Socket.IO events handled by the realtime gateway.",
		}, []string{"event", "result"}),
		socketConnectionsNow: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chess_backend_socket_connections",
			Help: "Current open Socket.IO connections.",
		}),
		roomCreationsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "chess_backend_room_creations_total",
			Help: "Rooms created grouped by mode, clock control and client profile.",
		}, []string{"mode", "clock_control", "device_type", "platform", "browser", "region"}),
		roomJoinsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "chess_backend_room_joins_total",
			Help: "Successful room joins grouped by role, mode and client profile.",
		}, []string{"role", "mode", "device_type", "platform", "browser", "region"}),
	}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) ObserveHTTPRequest(method string, path string, statusCode int, duration time.Duration) {
	normalizedPath := normalizeMetricsPath(path)
	m.httpRequestsTotal.WithLabelValues(method, normalizedPath, strconvStatus(statusCode)).Inc()
	m.httpRequestDuration.WithLabelValues(method, normalizedPath).Observe(duration.Seconds())
}

func (m *Metrics) ObserveSocketEvent(eventName string, result string) {
	m.socketEventsTotal.WithLabelValues(eventName, result).Inc()
}

func (m *Metrics) IncSocketConnections() {
	m.socketConnectionsNow.Inc()
}

func (m *Metrics) DecSocketConnections() {
	m.socketConnectionsNow.Dec()
}

func (m *Metrics) ObserveRoomCreated(mode GameMode, clockControl string, client *ClientTelemetry) {
	profile := normalizeClientTelemetry(client)
	m.roomCreationsTotal.WithLabelValues(
		normalizeMode(mode),
		normalizeClockControl(clockControl),
		profile.DeviceType,
		profile.Platform,
		profile.Browser,
		profile.Region,
	).Inc()
}

func (m *Metrics) ObserveRoomJoin(role ViewerRole, mode GameMode, client *ClientTelemetry) {
	profile := normalizeClientTelemetry(client)
	m.roomJoinsTotal.WithLabelValues(
		normalizeRole(role),
		normalizeMode(mode),
		profile.DeviceType,
		profile.Platform,
		profile.Browser,
		profile.Region,
	).Inc()
}

func normalizeMetricsPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/socket.io"):
		return "/socket.io"
	case strings.HasPrefix(path, "/api/"):
		return path
	case strings.HasPrefix(path, "/health/"):
		return path
	case path == "":
		return "/"
	default:
		return path
	}
}

func strconvStatus(code int) string {
	return strconv.Itoa(code)
}

func normalizeClientTelemetry(client *ClientTelemetry) ClientTelemetry {
	if client == nil {
		return ClientTelemetry{
			DeviceType: "unknown",
			Platform:   "unknown",
			Browser:    "unknown",
			Region:     "unknown",
		}
	}

	return ClientTelemetry{
		DeviceType: normalizeTelemetryValue(client.DeviceType),
		Platform:   normalizeTelemetryValue(client.Platform),
		Browser:    normalizeTelemetryValue(client.Browser),
		Region:     normalizeTelemetryValue(client.Region),
	}
}

func normalizeTelemetryValue(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "unknown"
	}
	return normalized
}

func normalizeMode(mode GameMode) string {
	if strings.TrimSpace(string(mode)) == "" {
		return "pvp"
	}
	return normalizeTelemetryValue(string(mode))
}

func normalizeClockControl(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "untimed":
		return "untimed"
	case "timed":
		return "timed"
	default:
		return "timed"
	}
}

func normalizeRole(role ViewerRole) string {
	if strings.TrimSpace(string(role)) == "" {
		return "unknown"
	}
	return normalizeTelemetryValue(string(role))
}
