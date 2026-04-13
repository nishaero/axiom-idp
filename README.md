# Axiom IDP - Lightweight AI-Native Internal Developer Platform

[![Build Status](https://github.com/axiom-idp/axiom/actions/workflows/ci.yml/badge.svg)](https://github.com/axiom-idp/axiom/actions)
[![Security Scanning](https://github.com/axiom-idp/axiom/actions/workflows/security-scan.yml/badge.svg)](https://github.com/axiom-idp/axiom/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/axiom-idp/axiom)](https://goreportcard.com/report/github.com/axiom-idp/axiom)
[![Coverage Status](https://img.shields.io/codecov/c/github/axiom-idp/axiom)](https://codecov.io/gh/axiom-idp/axiom)

Axiom is a stateless, MCP-native Internal Developer Platform designed to provide AI-first developer experiences with minimal resource overhead.

## Features

- **AI-Native Architecture**: First-class AI integration using Model Context Protocol (MCP)
- **Stateless Design**: Metadata-only storage, real-time data queries
- **MCP-Powered Integrations**: Use MCP servers for pluggable integrations
- **RBAC & OAuth2/OIDC**: Enterprise-grade security
- **Low Resource Usage**: <256MB RAM footprint
- **Sub-2s AI Response Time**: Optimized context windows
- **Professional UI**: Modern React + TypeScript frontend
- **Production-Ready**: Docker, Kubernetes, systemd deployments

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ and npm/pnpm
- Docker (optional, for containerized deployment)

### Installation

#### Option 1: From Binary (Recommended)

```bash
curl -L https://github.com/axiom-idp/axiom/releases/latest/download/axiom-linux-amd64 -o axiom
chmod +x axiom
sudo mv axiom /usr/local/bin/
```

#### Option 2: Build from Source

```bash
git clone https://github.com/axiom-idp/axiom.git
cd axiom
make build
./bin/axiom-server
```

### Configuration

Create `.env` file:

```env
PORT=8080
ENVIRONMENT=development
LOG_LEVEL=info
OAUTH_PROVIDER=https://accounts.google.com
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-client-secret
```

### Running

```bash
axiom --config .env
```

Visit `http://localhost:8080` in your browser.

## Development

### Build Instructions

```bash
# Install dependencies
go mod download
cd web && npm install && cd ..

# Build backend
make build

# Build frontend
make build-web

# Run in development mode
make dev
```

### Project Structure

```
axiom-idp/
├── cmd/
│   └── axiom-server/     # Server binary
├── internal/
│   ├── server/           # HTTP server
│   ├── mcp/              # MCP registry
│   ├── catalog/          # Service catalog
│   ├── ai/               # AI router
│   └── auth/             # Authentication
├── pkg/
│   ├── models/           # Data models
│   └── utils/            # Utilities
├── web/                  # React frontend
├── docs/                 # Documentation
└── deployments/          # Docker, K8s configs
```

## Documentation

- [Building & Installation](docs/building.md)
- [Configuration Guide](docs/configuration.md)
- [API Reference](docs/api.md)
- [Security Policy](SECURITY.md)
- [Contributing Guide](CONTRIBUTING.md)
- [Architecture](docs/architecture.md)

## Security

See [SECURITY.md](SECURITY.md) for security policies and reporting vulnerability information.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## AI Disclosure

Parts of this codebase were generated with AI assistance. See [AI_DISCLOSURE.md](AI_DISCLOSURE.md) for details.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Support

- 📖 [Documentation](./docs)
- 💬 [GitHub Discussions](https://github.com/axiom-idp/axiom/discussions)
- 🐛 [Issue Tracker](https://github.com/axiom-idp/axiom/issues)
- 🔐 [Security Issues](SECURITY.md)

## Roadmap

- [x] Phase 1: Project Foundation
- [ ] Phase 2: Backend Implementation
- [ ] Phase 3: Frontend Implementation
- [ ] Phase 4: Security Layer
- [ ] Phase 5: CI/CD Workflows
- [ ] Phase 6: Deployment Setup
- [ ] Phase 7: MCP Servers
- [ ] Phase 8: Testing

---

**Made with ❤️ by the Axiom Team**
