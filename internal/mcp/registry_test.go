package mcp

import (
	"testing"
)

func TestRegister(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register("k8s", "Kubernetes MCP", "mcp-kubernetes", []string{})
	if err != nil {
		t.Fatalf("Failed to register server: %v", err)
	}

	servers := registry.ListServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}
}

func TestRegisterDuplicate(t *testing.T) {
	registry := NewRegistry()

	registry.Register("k8s", "Kubernetes MCP", "mcp-kubernetes", []string{})

	err := registry.Register("k8s", "Kubernetes MCP", "mcp-kubernetes", []string{})
	if err != ErrServerAlreadyExists {
		t.Errorf("Expected ErrServerAlreadyExists, got %v", err)
	}
}

func TestGetServer(t *testing.T) {
	registry := NewRegistry()

	registry.Register("k8s", "Kubernetes MCP", "mcp-kubernetes", []string{})

	server, err := registry.GetServer("k8s")
	if err != nil {
		t.Fatalf("Failed to get server: %v", err)
	}

	if server.Name != "Kubernetes MCP" {
		t.Errorf("Expected name 'Kubernetes MCP', got '%s'", server.Name)
	}
}

func TestGetServerNotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.GetServer("nonexistent")
	if err != ErrServerNotFound {
		t.Errorf("Expected ErrServerNotFound, got %v", err)
	}
}
