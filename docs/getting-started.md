# Getting Started with Axiom IDP

## Prerequisites

- Go 1.22+
- Node.js 18+
- npm or pnpm
- Make
- Git

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/axiom-idp/axiom-idp.git
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
docker-compose up -d
```

## Configuration

Create `.env` file:

```bash
# Server
AXIOM_PORT=8080
AXIOM_ENV=development
AXIOM_LOG_LEVEL=info

# OAuth2 (optional)
AXIOM_OAUTH_PROVIDER=github
AXIOM_OAUTH_CLIENT_ID=your-client-id
AXIOM_OAUTH_CLIENT_SECRET=your-secret
```

Load and start:

```bash
source .env
axiom-server
```

## Next Steps

- [Configuration Guide](./configuration.md)
- [API Documentation](./api.md)
- [Architecture Overview](./architecture.md)
- [Deployment Guide](./deployment.md)
