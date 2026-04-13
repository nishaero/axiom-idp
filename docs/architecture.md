# Axiom IDP Architecture

## Overview

Axiom IDP is a stateless, MCP-native Internal Developer Platform with three main layers:

```
┌─────────────────────────────────┐
│    Frontend (React + TypeScript)  │
│  Web UI, Dashboard, AI Chat       │
└──────────────┬────────────────────┘
               │ HTTP/WebSocket
┌──────────────▼────────────────────┐
│   Backend (Go HTTP Server)        │
│  ├─ Router & Middleware           │
│  ├─ MCP Registry                  │
│  ├─ Catalog Index                 │
│  ├─ Auth Manager                  │
│  └─ AI Router                     │
└──────────────┬────────────────────┘
               │ Process/HTTP
    ┌──────────┴──────────┐
    │                     │
  ┌─▼──────────┐   ┌─────▼─────┐
  │ MCP Servers │   │External    │
  │ Kubernetes  │   │APIs        │
  │ Jenkins     │   │GitHub      │
  │ Custom...   │   │Slack, etc  │
  └─────────────┘   └────────────┘
```

## Key Components

### Frontend Layer
- **React 18** - Component framework
- **Vite** - Build tooling
- **TailwindCSS** - Styling
- **React Query** - Data fetching
- **Zustand** - State management
- **React Router** - Routing

### Backend Layer

#### HTTP Server
- Handles all client requests
- Routes to appropriate handlers
- Middleware chain (auth, logging, CORS)
- WebSocket support for real-time data

#### MCP Registry
- Manages MCP server lifecycle
- Process spawning and IPC
- Tool discovery and routing
- Resource management
- Hot-reloading support

#### Catalog Index
- Service metadata tracking
- Fast search and filtering
- Real-time updates from source systems
- No data duplication (pointers only)

#### Auth Manager
- OAuth2/OIDC flows
- Session management
- Token validation
- RBAC enforcement

#### AI Router
- Query optimization
- Context window management
- MCP tool selection
- Result formatting

## Data Flow

### Query Execution
1. User submits query via frontend
2. Auth middleware validates token
3. AI router optimizes context
4. MCP registry selects tools
5. Tools executed against source systems
6. Results aggregated and returned
7. Frontend displays results

### MCP Integration
1. Server registers MCP config
2. MCP processes spawn on demand
3. Tool calls via stdio/HTTP
4. Results streamed back
5. Process lifecycle managed

## Database

### Options
- **SQLite**: Development, small deployments (<100 users)
- **PostgreSQL**: Production, large scale

### Schema
- Users: Auth and profile
- Services: Catalog entries
- Audit Log: All operations
- API Keys: Auth tokens
- Sessions: User sessions

## Security Model

- **OAuth2/OIDC**: External authentication
- **API Keys**: Service-to-service auth
- **Sessions**: User sessions with rotation
- **RBAC**: Role-based access control
- **Audit Logging**: All operations logged
- **Rate Limiting**: Per-user and IP based
- **TLS**: All communication encrypted

## Performance Characteristics

- **Baseline Memory**: <256MB
- **Query Latency**: <2s (p95)
- **Context Token**: <2000 tokens average
- **Throughput**: 1000+ requests/seconds
- **Connections**: 100+ concurrent

## Extensibility

### Adding MCP Servers
1. Create MCP server (subprocess)
2. Add to `mcp.yaml` config
3. Implement required tools
4. Server auto-discovered on startup

### Custom Integrations
1. Implement HTTP API client
2. Register as tool endpoint
3. Return structured responses
4. Automatic catalog indexing

## Deployment Targets

- **Ubuntu systemd**: Single binary deployment
- **Docker**: Containerized deployment
- **Docker Compose**: Local development
- **Kubernetes**: Production scale
