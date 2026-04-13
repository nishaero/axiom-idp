package mcp

import "errors"

var (
	ErrServerNotFound      = errors.New("MCP server not found")
	ErrServerAlreadyExists = errors.New("MCP server already registered")
	ErrServerNotRunning    = errors.New("MCP server is not running")
	ErrToolNotFound        = errors.New("tool not found on MCP server")
)
