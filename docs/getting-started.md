# Getting Started with Axiom IDP

## Prerequisites

- Go 1.24+
- Node.js 24+
- npm
- Docker and Docker Compose v2
- Make
- Git

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/nishaero/axiom-idp.git
cd axiom-idp

# Install dependencies
go mod download
cd web && npm install && cd ..

# Build
make build-all

# Run development
make dev
```

### Docker

```bash
docker compose up -d --build
```

## Configuration

Create `.env` file:

```bash
# Server
AXIOM_HOST=0.0.0.0
AXIOM_PORT=8081
AXIOM_ENV=development
AXIOM_LOG_LEVEL=info
AXIOM_SESSION_SECRET=replace-with-a-long-random-secret

# AI
AXIOM_AI_BACKEND=local
# AXIOM_AI_BACKEND=ollama
# AXIOM_AI_BASE_URL=http://host.docker.internal:11434
# AXIOM_AI_MODEL=qwen3.5:9b

# Optional shared runtime state
# AXIOM_DB_DRIVER=postgres
# AXIOM_DB_URL=postgres://postgres.default.svc.cluster.local:5432/axiom?sslmode=disable
```

Load and start:

```bash
source .env
./bin/axiom-server
```

## Next Steps

- [README](../README.md)
- [Deployment Guide](../DEPLOYMENT.md)
- [Architecture Overview](./architecture.md)
