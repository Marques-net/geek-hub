package login

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
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	loginRecordsTotal   *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		httpRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "geek_hub_login_service_http_requests_total",
			Help: "HTTP requests handled by the login service.",
		}, []string{"method", "path", "status_code"}),
		httpRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "geek_hub_login_service_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds for the login service.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		loginRecordsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "geek_hub_login_service_login_records_total",
			Help: "Login persistence attempts grouped by provider and result.",
		}, []string{"provider", "result"}),
	}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) ObserveHTTPRequest(method string, path string, statusCode int, duration time.Duration) {
	normalizedPath := normalizeMetricsPath(path)
	m.httpRequestsTotal.WithLabelValues(method, normalizedPath, strconv.Itoa(statusCode)).Inc()
	m.httpRequestDuration.WithLabelValues(method, normalizedPath).Observe(duration.Seconds())
}

func (m *Metrics) ObserveLoginRecord(provider string, result string) {
	m.loginRecordsTotal.WithLabelValues(normalizeValue(provider), normalizeValue(result)).Inc()
}

func normalizeMetricsPath(path string) string {
	switch {
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

func normalizeValue(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "unknown"
	}

	return normalized
}
