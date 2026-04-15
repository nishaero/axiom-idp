package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type statusCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type statusAlert struct {
	Severity string `json:"severity"`
	Scope    string `json:"scope"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

type serviceStatusSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	ReleaseState string `json:"release_state"`
	RiskLevel    string `json:"risk_level"`
	HealthScore  int    `json:"health_score"`
}

type platformStatus struct {
	Status             string                 `json:"status"`
	Environment        string                 `json:"environment"`
	StartedAt          time.Time              `json:"started_at"`
	Uptime             string                 `json:"uptime"`
	AIBackend          string                 `json:"ai_backend"`
	KubernetesNS       string                 `json:"kubernetes_namespace"`
	Checks             []statusCheck          `json:"checks"`
	Alerts             []statusAlert          `json:"alerts"`
	Overview           catalogOverview        `json:"overview"`
	Portfolio          portfolioIntelligence  `json:"portfolio"`
	Services           []serviceStatusSummary `json:"services"`
	RecentAudit        []AuditLog             `json:"recent_audit"`
	Audit              AuditStats             `json:"audit"`
	RateLimiting       RateLimiterStats       `json:"rate_limiting"`
	ObservabilityNotes []string               `json:"observability_notes"`
}

type observabilityEndpoint struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type observabilityResponse struct {
	Platform              platformStatus          `json:"platform"`
	Telemetry             telemetrySnapshot       `json:"telemetry"`
	Endpoints             []observabilityEndpoint `json:"endpoints"`
	MetricsEndpoint       string                  `json:"metrics_endpoint"`
	PrometheusAnnotations []string                `json:"prometheus_annotations"`
	Notes                 []string                `json:"notes"`
}

func (s *Server) buildPlatformStatus() platformStatus {
	startedAt := s.startedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	checks := []statusCheck{
		{
			Name:    "auth",
			Status:  statusReady,
			Message: "Signed session tokens and RBAC middleware are active.",
		},
		{
			Name:    "rate_limiting",
			Status:  statusReady,
			Message: fmt.Sprintf("Token bucket rate limiting is configured at %d requests per %s.", s.config.RateLimitRequests, s.config.RateLimitWindow),
		},
		{
			Name:    "ai_backend",
			Status:  statusReady,
			Message: fmt.Sprintf("AI backend '%s' is configured for assistant workflows.", s.config.AIBackend),
		},
		{
			Name:    "delivery",
			Status:  statusReady,
			Message: "Direct Kubernetes deployment and GitOps delivery paths are available.",
		},
	}

	alerts := make([]statusAlert, 0, 4)
	for _, service := range demoCatalog {
		if strings.EqualFold(service.Owner, "Unassigned") {
			alerts = append(alerts, statusAlert{
				Severity: "high",
				Scope:    service.Name,
				Title:    "Ownership gap",
				Detail:   "Service has no accountable owner and should stay blocked for production release.",
			})
		}
		if service.ReleaseState == "blocked" || service.Status == "degraded" {
			alerts = append(alerts, statusAlert{
				Severity: "medium",
				Scope:    service.Name,
				Title:    "Operational attention required",
				Detail:   fmt.Sprintf("%s is %s with release state %s.", service.Name, service.Status, service.ReleaseState),
			})
		}
	}

	if len(alerts) == 0 {
		alerts = append(alerts, statusAlert{
			Severity: "info",
			Scope:    "platform",
			Title:    "No active blocking alerts",
			Detail:   "Core release and compliance indicators are currently within expected bounds.",
		})
	}

	services := make([]serviceStatusSummary, 0, len(demoCatalog))
	for _, service := range demoCatalog {
		services = append(services, serviceStatusSummary{
			ID:           service.ID,
			Name:         service.Name,
			Status:       service.Status,
			ReleaseState: service.ReleaseState,
			RiskLevel:    service.RiskLevel,
			HealthScore:  service.HealthScore,
		})
	}

	status, readinessChecks := s.buildReadinessStatus()
	checks = append(checks, readinessChecks...)
	if status == statusReady {
		for _, alert := range alerts {
			if alert.Severity == "high" {
				status = statusDegraded
				break
			}
		}
	}

	return platformStatus{
		Status:       status,
		Environment:  s.config.Environment,
		StartedAt:    startedAt,
		Uptime:       time.Since(startedAt).Round(time.Second).String(),
		AIBackend:    s.config.AIBackend,
		KubernetesNS: s.config.KubernetesNamespace,
		Checks:       checks,
		Alerts:       alerts,
		Overview:     catalogSummary(demoCatalog),
		Portfolio:    buildPortfolioIntelligence(demoCatalog),
		Services:     services,
		RecentAudit:  s.auditor.GetLogs("", 12),
		Audit:        s.auditor.Stats(),
		RateLimiting: s.rateLimiter.Stats(),
		ObservabilityNotes: []string{
			"Dashboard values marked live come from the backend platform status endpoint.",
			"Prometheus and OpenTelemetry exporters are the next recommended step for production telemetry depth.",
			"Release blockers are derived from ownership, evidence, and service health signals.",
		},
	}
}

func (s *Server) handleObservability(w http.ResponseWriter, r *http.Request) {
	platform := s.buildPlatformStatus()
	if s.metrics != nil {
		s.metrics.UpdatePlatformSnapshot(platform)
		s.metrics.UpdateOperationalSnapshots(platform.Audit, platform.RateLimiting)
	}

	response := observabilityResponse{
		Platform:        platform,
		Telemetry:       telemetrySnapshot{},
		Endpoints:       buildObservabilityEndpoints(platform.Status),
		MetricsEndpoint: "/metrics",
		PrometheusAnnotations: []string{
			"prometheus.io/scrape=true",
			"prometheus.io/path=/metrics",
			"prometheus.io/port=80",
		},
		Notes: []string{
			"Prometheus scrapes the /metrics endpoint directly; the UI shows a compact status snapshot from the control plane.",
			"Readiness and liveness are exposed separately so minikube and production deployments can distinguish startup from health.",
			"Telemetry counters are stored in-process for local and demo deployments; use shared storage if you need multi-replica durability.",
		},
	}
	if s.metrics != nil {
		response.Telemetry = s.metrics.Snapshot()
	}

	writeJSON(w, http.StatusOK, response)
}

func buildObservabilityEndpoints(platformState string) []observabilityEndpoint {
	endpointStatus := func(status string) string {
		switch status {
		case statusReady:
			return "healthy"
		case statusDegraded:
			return "degraded"
		case statusUnready:
			return "unhealthy"
		default:
			return "unknown"
		}
	}

	platformTone := endpointStatus(platformState)
	return []observabilityEndpoint{
		{
			Name:        "Live probe",
			Path:        "/live",
			Status:      "healthy",
			Description: "Container liveness probe for Kubernetes and Docker entrypoints.",
		},
		{
			Name:        "Readiness probe",
			Path:        "/ready",
			Status:      platformTone,
			Description: "Checks session secret and AI backend configuration before serving traffic.",
		},
		{
			Name:        "Health snapshot",
			Path:        "/health",
			Status:      platformTone,
			Description: "Returns the platform status summary used by the dashboard and smoke tests.",
		},
		{
			Name:        "Prometheus metrics",
			Path:        "/metrics",
			Status:      "healthy",
			Description: "Prometheus exposition for HTTP, AI, deployment, audit, and rate-limit telemetry.",
		},
		{
			Name:        "Platform status API",
			Path:        "/api/v1/platform/status",
			Status:      platformTone,
			Description: "Structured control-plane snapshot consumed by the dashboard and operator workflows.",
		},
	}
}

const (
	statusReady    = "ready"
	statusDegraded = "degraded"
	statusUnready  = "unready"
)

func (s *Server) buildReadinessStatus() (string, []statusCheck) {
	checks := make([]statusCheck, 0, 2)

	if strings.TrimSpace(s.config.SessionSecret) == "" {
		checks = append(checks, statusCheck{
			Name:    "session_secret",
			Status:  statusUnready,
			Message: "Session secret is not configured.",
		})
		return statusUnready, checks
	}

	checks = append(checks, statusCheck{
		Name:    "session_secret",
		Status:  statusReady,
		Message: "Session secret is configured.",
	})

	if strings.TrimSpace(s.config.AIBackend) == "" {
		checks = append(checks, statusCheck{
			Name:    "ai_backend",
			Status:  statusUnready,
			Message: "AI backend is not configured.",
		})
		return statusUnready, checks
	}

	checks = append(checks, statusCheck{
		Name:    "ai_backend",
		Status:  statusReady,
		Message: "AI backend configuration is present.",
	})

	return statusReady, checks
}
