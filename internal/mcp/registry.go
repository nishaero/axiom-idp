package mcp

import (
	"context"
	"sync"
)

// Server represents a single MCP server instance
type Server struct {
	ID      string
	Name    string
	Command string
	Args    []string
	running bool
	mu      sync.RWMutex
}

// Registry manages all MCP servers
type Registry struct {
	servers map[string]*Server
	mu      sync.RWMutex
}

// NewRegistry creates a new MCP registry
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]*Server),
	}
}

// Register adds a new MCP server to the registry
func (r *Registry) Register(id, name, command string, args []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[id]; exists {
		return ErrServerAlreadyExists
	}

	r.servers[id] = &Server{
		ID:      id,
		Name:    name,
		Command: command,
		Args:    args,
		running: false,
	}

	return nil
}

// GetServer retrieves a registered MCP server
func (r *Registry) GetServer(id string) (*Server, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	server, exists := r.servers[id]
	if !exists {
		return nil, ErrServerNotFound
	}

	return server, nil
}

// ListServers returns all registered servers
func (r *Registry) ListServers() []*Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]*Server, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, server)
	}

	return servers
}

// Start initializes and starts an MCP server
func (r *Registry) Start(ctx context.Context, serverID string) error {
	server, err := r.GetServer(serverID)
	if err != nil {
		return err
	}

	server.mu.Lock()
	server.running = true
	server.mu.Unlock()

	// TODO: Implement actual process spawning

	return nil
}

// Stop shuts down an MCP server
func (r *Registry) Stop(ctx context.Context, serverID string) error {
	server, err := r.GetServer(serverID)
	if err != nil {
		return err
	}

	server.mu.Lock()
	server.running = false
	server.mu.Unlock()

	// TODO: Implement actual process termination

	return nil
}

// CallTool executes a tool on an MCP server
func (r *Registry) CallTool(ctx context.Context, serverID, toolName string, args map[string]interface{}) (interface{}, error) {
	server, err := r.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	server.mu.RLock()
	if !server.running {
		server.mu.RUnlock()
		return nil, ErrServerNotRunning
	}
	server.mu.RUnlock()

	// TODO: Implement actual tool invocation

	return nil, nil
}
