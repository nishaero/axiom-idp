# MCP Servers

Model Context Protocol (MCP) server implementations for Axiom IDP.

## Kubernetes MCP Server

Provides tools for Kubernetes cluster management and monitoring:
- `get_pods` - List pods in a namespace
- `get_services` - List services in a namespace
- `describe_pod` - Get detailed pod information

### Building

```bash
cd kubernetes
go build -o mcp-kubernetes main.go
```

### Usage

```bash
./mcp-kubernetes
```

## GitHub MCP Server

Provides tools for GitHub repository integration:
- `get_repos` - List repositories for an owner
- `get_issues` - List issues in a repository
- `get_pull_requests` - List pull requests

### Building

```bash
cd github
go build -o mcp-github main.go
```

### Usage

Set GitHub token:
```bash
export GITHUB_TOKEN=your-token
./mcp-github
```

## Creating Custom MCP Servers

To create a new MCP server:

1. Create a directory: `mkdir mcp-servers/myserver`
2. Implement JSON-RPC interface
3. Handle `list_tools` and `call_tool` methods
4. Register in main Axiom configuration

See templates in this directory for examples.
