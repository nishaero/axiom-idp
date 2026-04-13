package models

import "time"

// Service represents a deployed service
type Service struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Owner       string                 `json:"owner"`
	Repository  string                 `json:"repository"`
	Language    string                 `json:"language"`
	Status      string                 `json:"status"`
	Tags        []string               `json:"tags"`
	Meta        map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Deployment represents a service deployment
type Deployment struct {
	ID        string    `json:"id"`
	ServiceID string    `json:"service_id"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKey represents an API authentication key
type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Key       string    `json:"key"`
	Secret    string    `json:"secret"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}
