package server

import (
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type telemetrySnapshot struct {
	HTTPRequestsTotal       int64  `json:"http_requests_total"`
	HTTPErrorResponsesTotal int64  `json:"http_error_responses_total"`
	HTTPRateLimitedTotal    int64  `json:"http_rate_limited_total"`
	AIRequestsTotal         int64  `json:"ai_requests_total"`
	AIFailuresTotal         int64  `json:"ai_failures_total"`
	DeploymentRequestsTotal int64  `json:"deployment_requests_total"`
	AuditEventsTotal        int64  `json:"audit_events_total"`
	LastRequestAt           string `json:"last_request_at,omitempty"`
	LastAIRequestAt         string `json:"last_ai_request_at,omitempty"`
	LastDeploymentAt        string `json:"last_deployment_at,omitempty"`
	LastAuditAt             string `json:"last_audit_at,omitempty"`
}

type Metrics struct {
	registry *prometheus.Registry

	httpRequestsTotal       *prometheus.CounterVec
	httpRequestDuration     *prometheus.HistogramVec
	aiRequestsTotal         *prometheus.CounterVec
	aiRequestDuration       *prometheus.HistogramVec
	deploymentRequestsTotal *prometheus.CounterVec
	deploymentDuration      *prometheus.HistogramVec
	auditEventsTotal        *prometheus.CounterVec
	rateLimitedTotal        prometheus.Counter
	platformState           *prometheus.GaugeVec
	platformOverview        *prometheus.GaugeVec
	rateLimiterOverview     *prometheus.GaugeVec
	auditOverview           *prometheus.GaugeVec

	httpRequestsSeen       atomic.Int64
	httpErrorsSeen         atomic.Int64
	httpRateLimitedSeen    atomic.Int64
	aiRequestsSeen         atomic.Int64
	aiFailuresSeen         atomic.Int64
	deploymentRequestsSeen atomic.Int64
	auditEventsSeen        atomic.Int64
	lastRequestAt          atomic.Int64
	lastAIRequestAt        atomic.Int64
	lastDeploymentAt       atomic.Int64
	lastAuditAt            atomic.Int64
}

func newMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "axiom_http_requests_total",
				Help: "Total number of HTTP requests observed by Axiom.",
			},
			[]string{"method", "route", "status_code", "outcome"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "axiom_http_request_duration_seconds",
				Help:    "HTTP request durations observed by Axiom.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),
		aiRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "axiom_ai_requests_total",
				Help: "Total number of AI and AI-assisted requests handled by Axiom.",
			},
			[]string{"backend", "intent", "outcome"},
		),
		aiRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "axiom_ai_request_duration_seconds",
				Help:    "Duration of AI and AI-assisted requests handled by Axiom.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"backend", "intent"},
		),
		deploymentRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "axiom_deployment_requests_total",
				Help: "Total number of deployment and infrastructure actions handled by Axiom.",
			},
			[]string{"provider", "action", "outcome"},
		),
		deploymentDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "axiom_deployment_duration_seconds",
				Help:    "Duration of deployment and infrastructure actions handled by Axiom.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider", "action"},
		),
		auditEventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "axiom_audit_events_total",
				Help: "Total number of audit events emitted by Axiom.",
			},
			[]string{"status"},
		),
		rateLimitedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "axiom_rate_limited_requests_total",
				Help: "Total number of requests rejected by the rate limiter.",
			},
		),
		platformState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "axiom_platform_state",
				Help: "Current platform readiness state exposed as a labeled gauge.",
			},
			[]string{"state"},
		),
		platformOverview: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "axiom_platform_overview",
				Help: "High-level platform overview values mirrored from the control-plane status.",
			},
			[]string{"metric"},
		),
		rateLimiterOverview: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "axiom_rate_limiter_overview",
				Help: "Current rate limiter state mirrored from the control-plane status.",
			},
			[]string{"metric"},
		),
		auditOverview: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "axiom_audit_overview",
				Help: "Current audit trail summary mirrored from the control-plane status.",
			},
			[]string{"metric"},
		),
	}

	registry.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.aiRequestsTotal,
		m.aiRequestDuration,
		m.deploymentRequestsTotal,
		m.deploymentDuration,
		m.auditEventsTotal,
		m.rateLimitedTotal,
		m.platformState,
		m.platformOverview,
		m.rateLimiterOverview,
		m.auditOverview,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "metrics unavailable", http.StatusServiceUnavailable)
		})
	}

	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		route := requestRouteLabel(r)
		statusCode := strconv.Itoa(recorder.statusCode)
		outcome := metricOutcome(recorder.statusCode)

		m.httpRequestsTotal.WithLabelValues(r.Method, route, statusCode, outcome).Inc()
		m.httpRequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		m.httpRequestsSeen.Add(1)
		if recorder.statusCode >= http.StatusInternalServerError {
			m.httpErrorsSeen.Add(1)
		}
		m.lastRequestAt.Store(time.Now().Unix())
	})
}

func (m *Metrics) RecordAIRequest(backend, intent string, duration time.Duration, outcome string) {
	if m == nil {
		return
	}

	backend = strings.TrimSpace(backend)
	if backend == "" {
		backend = "unknown"
	}
	intent = strings.TrimSpace(intent)
	if intent == "" {
		intent = "unknown"
	}
	outcome = strings.TrimSpace(outcome)
	if outcome == "" {
		outcome = "success"
	}

	m.aiRequestsTotal.WithLabelValues(backend, intent, outcome).Inc()
	m.aiRequestDuration.WithLabelValues(backend, intent).Observe(duration.Seconds())
	m.aiRequestsSeen.Add(1)
	if outcome != "success" {
		m.aiFailuresSeen.Add(1)
	}
	m.lastAIRequestAt.Store(time.Now().Unix())
}

func (m *Metrics) RecordDeploymentRequest(provider, action string, duration time.Duration, outcome string) {
	if m == nil {
		return
	}

	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "unknown"
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "unknown"
	}
	outcome = strings.TrimSpace(outcome)
	if outcome == "" {
		outcome = "success"
	}

	m.deploymentRequestsTotal.WithLabelValues(provider, action, outcome).Inc()
	m.deploymentDuration.WithLabelValues(provider, action).Observe(duration.Seconds())
	m.deploymentRequestsSeen.Add(1)
	m.lastDeploymentAt.Store(time.Now().Unix())
}

func (m *Metrics) RecordAudit(status string) {
	if m == nil {
		return
	}

	status = strings.TrimSpace(status)
	if status == "" {
		status = "success"
	}

	m.auditEventsTotal.WithLabelValues(status).Inc()
	m.auditEventsSeen.Add(1)
	m.lastAuditAt.Store(time.Now().Unix())
}

func (m *Metrics) RecordRateLimit() {
	if m == nil {
		return
	}

	m.rateLimitedTotal.Inc()
	m.httpRateLimitedSeen.Add(1)
	m.lastRequestAt.Store(time.Now().Unix())
}

func (m *Metrics) UpdatePlatformSnapshot(status platformStatus) {
	if m == nil {
		return
	}

	for _, state := range []string{statusReady, statusDegraded, statusUnready} {
		m.platformState.WithLabelValues(state).Set(0)
	}
	m.platformState.WithLabelValues(status.Status).Set(1)

	setMetric := func(name string, value float64) {
		m.platformOverview.WithLabelValues(name).Set(value)
	}

	setMetric("total_services", float64(status.Overview.TotalServices))
	setMetric("ready_services", float64(status.Overview.ReadyServices))
	setMetric("watch_services", float64(status.Overview.WatchServices))
	setMetric("blocked_services", float64(status.Overview.BlockedServices))
	setMetric("owner_gap_count", float64(status.Overview.OwnerGapCount))
	setMetric("release_readiness", float64(status.Overview.ReleaseReadiness))
	setMetric("evidence_coverage", float64(status.Overview.EvidenceCoverage))
	setMetric("portfolio_total_services", float64(status.Portfolio.TotalServices))
	setMetric("portfolio_ready_services", float64(status.Portfolio.ReadyCount))
	setMetric("portfolio_watch_services", float64(status.Portfolio.WatchCount))
	setMetric("portfolio_blocked_services", float64(status.Portfolio.BlockedCount))
	setMetric("audit_entries", float64(status.Audit.Entries))
	setMetric("audit_error_count", float64(status.Audit.ErrorCount))
	setMetric("audit_denied_count", float64(status.Audit.DeniedCount))
	setMetric("rate_limit_tracked_keys", float64(status.RateLimiting.TrackedKeys))
	setMetric("rate_limit_requests_seen", float64(status.RateLimiting.RequestsSeen))
	setMetric("rate_limit_requests_per_min", float64(status.RateLimiting.RequestsPerMin))
	setMetric("uptime_seconds", time.Since(status.StartedAt).Seconds())
}

func (m *Metrics) UpdateOperationalSnapshots(audit AuditStats, rateLimiter RateLimiterStats) {
	if m == nil {
		return
	}

	setMetric := func(gauge *prometheus.GaugeVec, name string, value float64) {
		gauge.WithLabelValues(name).Set(value)
	}

	setMetric(m.auditOverview, "entries", float64(audit.Entries))
	setMetric(m.auditOverview, "error_count", float64(audit.ErrorCount))
	setMetric(m.auditOverview, "denied_count", float64(audit.DeniedCount))
	setMetric(m.auditOverview, "success_count", float64(audit.SuccessCount))

	setMetric(m.rateLimiterOverview, "enabled", boolToFloat(rateLimiter.Enabled))
	setMetric(m.rateLimiterOverview, "tracked_keys", float64(rateLimiter.TrackedKeys))
	setMetric(m.rateLimiterOverview, "requests_seen", float64(rateLimiter.RequestsSeen))
	setMetric(m.rateLimiterOverview, "requests_per_min", float64(rateLimiter.RequestsPerMin))
}

func (m *Metrics) Snapshot() telemetrySnapshot {
	if m == nil {
		return telemetrySnapshot{}
	}

	return telemetrySnapshot{
		HTTPRequestsTotal:       m.httpRequestsSeen.Load(),
		HTTPErrorResponsesTotal: m.httpErrorsSeen.Load(),
		HTTPRateLimitedTotal:    m.httpRateLimitedSeen.Load(),
		AIRequestsTotal:         m.aiRequestsSeen.Load(),
		AIFailuresTotal:         m.aiFailuresSeen.Load(),
		DeploymentRequestsTotal: m.deploymentRequestsSeen.Load(),
		AuditEventsTotal:        m.auditEventsSeen.Load(),
		LastRequestAt:           formatUnixTime(m.lastRequestAt.Load()),
		LastAIRequestAt:         formatUnixTime(m.lastAIRequestAt.Load()),
		LastDeploymentAt:        formatUnixTime(m.lastDeploymentAt.Load()),
		LastAuditAt:             formatUnixTime(m.lastAuditAt.Load()),
	}
}

func requestRouteLabel(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if template, err := route.GetPathTemplate(); err == nil && strings.TrimSpace(template) != "" {
			return template
		}
	}

	if r.URL != nil && strings.TrimSpace(r.URL.Path) != "" {
		return r.URL.Path
	}

	return "unknown"
}

func metricOutcome(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "error"
	case statusCode >= 400:
		return "client_error"
	default:
		return "success"
	}
}

func boolToFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func formatUnixTime(value int64) string {
	if value <= 0 {
		return ""
	}

	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}
