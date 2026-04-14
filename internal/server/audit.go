package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/axiom-idp/axiom/internal/auth"
)

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Status    string                 `json:"status"`
	Details   map[string]interface{} `json:"details"`
	CreatedAt time.Time              `json:"created_at"`
	Error     string                 `json:"error,omitempty"`
}

// Auditor logs all server actions.
type Auditor struct {
	mu   sync.RWMutex
	logs []AuditLog
}

type AuditStats struct {
	Entries      int       `json:"entries"`
	LastEntryAt  time.Time `json:"last_entry_at,omitempty"`
	ErrorCount   int       `json:"error_count"`
	DeniedCount  int       `json:"denied_count"`
	SuccessCount int       `json:"success_count"`
}

// NewAuditor creates a new auditor.
func NewAuditor() *Auditor {
	return &Auditor{
		logs: make([]AuditLog, 0, 256),
	}
}

// Log records an audit log entry.
func (a *Auditor) Log(ctx context.Context, userID, action, resource, status string, details map[string]interface{}) {
	if a == nil {
		return
	}

	entry := AuditLog{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Status:    status,
		Details:   cloneDetails(details),
		CreatedAt: time.Now().UTC(),
	}

	a.mu.Lock()
	a.logs = append(a.logs, entry)
	a.mu.Unlock()
}

// LogError records an error in audit log.
func (a *Auditor) LogError(ctx context.Context, userID, action, resource string, err error) {
	if a == nil {
		return
	}

	entry := AuditLog{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Status:    "error",
		CreatedAt: time.Now().UTC(),
		Error:     err.Error(),
	}

	a.mu.Lock()
	a.logs = append(a.logs, entry)
	a.mu.Unlock()
}

// GetLogs returns audit logs with optional filtering.
func (a *Auditor) GetLogs(userID string, limit int) []AuditLog {
	if a == nil {
		return nil
	}
	if limit <= 0 {
		limit = 100
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	filtered := make([]AuditLog, 0, len(a.logs))
	for _, log := range a.logs {
		if userID == "" || log.UserID == userID {
			filtered = append(filtered, cloneAuditLog(log))
		}
	}

	if len(filtered) > limit {
		return append([]AuditLog(nil), filtered[len(filtered)-limit:]...)
	}

	return filtered
}

// Middleware records request audit entries.
func (a *Auditor) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			next.ServeHTTP(w, r)
			return
		}

		recorder := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(recorder, r)

		userID := auth.UserIDFromContext(r.Context())
		statusLabel := "success"
		switch {
		case recorder.statusCode >= http.StatusInternalServerError:
			statusLabel = "error"
		case recorder.statusCode == http.StatusUnauthorized || recorder.statusCode == http.StatusForbidden:
			statusLabel = "denied"
		}

		details := map[string]interface{}{
			"method":       r.Method,
			"path":         r.URL.Path,
			"remote_addr":  r.RemoteAddr,
			"status_code":  recorder.statusCode,
			"duration_ms":  time.Since(start).Milliseconds(),
			"user_agent":   r.UserAgent(),
			"content_type": r.Header.Get("Content-Type"),
			"request_id":   r.Header.Get("X-Request-ID"),
		}
		a.Log(r.Context(), userID, "http_request", r.URL.Path, statusLabel, details)
	})
}

func (a *Auditor) Stats() AuditStats {
	if a == nil {
		return AuditStats{}
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := AuditStats{Entries: len(a.logs)}
	for _, entry := range a.logs {
		if entry.CreatedAt.After(stats.LastEntryAt) {
			stats.LastEntryAt = entry.CreatedAt
		}

		switch entry.Status {
		case "error":
			stats.ErrorCount++
		case "denied":
			stats.DeniedCount++
		default:
			stats.SuccessCount++
		}
	}

	return stats
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *auditResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func cloneDetails(details map[string]interface{}) map[string]interface{} {
	if len(details) == 0 {
		return map[string]interface{}{}
	}

	out := make(map[string]interface{}, len(details))
	for key, value := range details {
		out[key] = value
	}
	return out
}

func cloneAuditLog(log AuditLog) AuditLog {
	log.Details = cloneDetails(log.Details)
	return log
}
