package catalog

import "time"

// DiscoveryConfig is a minimal configuration for service discovery.
type DiscoveryConfig struct {
	EnableDocker      bool
	EnableKubernetes  bool
	EnableEventBus    bool
	EnableRedisStream bool
	EnableWebSocket   bool
	EnableHotReload   bool
}

// ServiceStatus represents the health status of a service.
type ServiceStatus string

const (
	StatusRunning    ServiceStatus = "running"
	StatusStarting   ServiceStatus = "starting"
	StatusStopped    ServiceStatus = "stopped"
	StatusFailed     ServiceStatus = "failed"
	StatusUnhealthy  ServiceStatus = "unhealthy"
	StatusUnknown    ServiceStatus = "unknown"
	StatusRestarting ServiceStatus = "restarting"
)

// ServiceSummary is a minimal record used by discovery helpers.
type ServiceSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}
