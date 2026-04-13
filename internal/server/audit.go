package server

import (
	"context"
	"fmt"
	"time"
)

// AuditLog represents an audit log entry
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

// Auditor logs all server actions
type Auditor struct {
	logs   []AuditLog
}

// NewAuditor creates a new auditor
func NewAuditor() *Auditor {
	return &Auditor{
		logs: make([]AuditLog, 0),
	}
}

// Log records an audit log entry
func (a *Auditor) Log(ctx context.Context, userID, action, resource, status string, details map[string]interface{}) {
	logEntry := AuditLog{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Status:    status,
		Details:   details,
		CreatedAt: time.Now(),
	}

	a.logs = append(a.logs, logEntry)
}

// LogError records an error in audit log
func (a *Auditor) LogError(ctx context.Context, userID, action, resource string, err error) {
	logEntry := AuditLog{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Status:    "error",
		CreatedAt: time.Now(),
		Error:     err.Error(),
	}

	a.logs = append(a.logs, logEntry)
}

// GetLogs returns audit logs with optional filtering
func (a *Auditor) GetLogs(userID string, limit int) []AuditLog {
	if limit == 0 {
		limit = 100
	}

	var filtered []AuditLog
	for _, log := range a.logs {
		if userID == "" || log.UserID == userID {
			filtered = append(filtered, log)
		}
	}

	// Return last 'limit' entries
	if len(filtered) > limit {
		return filtered[len(filtered)-limit:]
	}

	return filtered
}
