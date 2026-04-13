# Axiom IDP - AI-Native Internal Developer Platform

[![Build Status](https://github.com/axiom-idp/axiom/actions/workflows/ci.yml/badge.svg)](https://github.com/axiom-idp/axiom/actions)
[![Security Scanning](https://github.com/axiom-idp/axiom/actions/workflows/security-scan.yml/badge.svg)](https://github.com/axiom-idp/axiom/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/axiom-idp/axiom)](https://goreportcard.com/report/github.com/axiom-idp/axiom)
[![Code Coverage](https://codecov.io/github/axiom-idp/axiom/graph/badge.svg)](https://codecov.io/github/axiom-idp/axiom)

<div align="center">
  <img src="docs/assets/axiom-logo.svg" alt="Axiom IDP Logo" width="200" style="border-radius: 12px;" />
  <br>
  <strong>Stateless, MCP-Native Internal Developer Platform with AI-First Design</strong>
  <br>
  <a href="https://github.com/axiom-idp/axiom/releases/latest">Download Latest Release</a>
  <br><br>
  <a href="SECURITY.md"><img src="https://img.shields.io/badge/B%2BSI%20C5%20Compliant-Compliant-green" alt="BSI C5 Compliant"></a>
  <a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/badge/License-Apache%202.0-blue" alt="License"></a>
</div>

---

## 🎯 Overview

Axiom is a stateless, MCP-native Internal Developer Platform designed to provide AI-first developer experiences with minimal resource overhead. Built with security and performance at its core, Axiom leverages the Model Context Protocol (MCP) for pluggable integrations while maintaining enterprise-grade security standards.

### ✨ Key Features

- **AI-Native Architecture**: First-class AI integration using Model Context Protocol (MCP)
- **Stateless Design**: Metadata-only storage, real-time data queries via Redis
- **MCP-Powered Integrations**: Use MCP servers for pluggable integrations (GitHub, Kubernetes, etc.)
- **RBAC & OAuth2/OIDC**: Enterprise-grade security with fine-grained access control
- **Low Resource Usage**: <256MB RAM footprint, optimized for edge deployment
- **Sub-2s AI Response Time**: Optimized context windows for rapid insights
- **Professional UI**: Modern React + TypeScript + Tailwind CSS frontend
- **Production-Ready**: Docker, Kubernetes, systemd deployments included
- **Security First**: BSI C5 compliant, comprehensive vulnerability scanning
- **Comprehensive Logging**: Audit trails, health monitoring, metrics export

---

## 📥 Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ and npm/pnpm
- Docker (optional, for containerized deployment)
- Redis 7+ (for metadata storage)

### Installation

#### Option 1: From GitHub Release (Recommended)

```bash
# Download latest release
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

Create `.env` file in the root directory:

```env
# Server Configuration
PORT=8080
ENVIRONMENT=development
LOG_LEVEL=info

# MCP Configuration
MCP_SERVERS=github,kubernetes,terraform

# OAuth2 Configuration
OAUTH_PROVIDER=https://accounts.google.com
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-client-secret

# Redis Configuration
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your-redis-password
```

### Running

```bash
# Start the server
./bin/axiom-server

# Or with Docker
docker compose up -d

# Or with Docker Compose (full stack)
docker-compose up -d
```

Visit `http://localhost:8080` in your browser.

---

## 🏗️ Development

### Build Instructions

```bash
# Install dependencies
go mod download
cd web && npm install && cd ..

# Build backend binary
make build

# Build frontend
cd web && npm run build && cd ..

# Run all targets
make build-all

# Run in development mode
make dev
```

### Docker Development

```bash
docker compose --profile dev up
```

### Project Structure

```
axiom-idp/
├── cmd/                    # Application entry points
│   └── axiom-server/      # Server binary
├── internal/              # Internal packages
│   ├── server/           # HTTP server & routing
│   ├── mcp/              # MCP registry & servers
│   ├── catalog/          # Service catalog
│   ├── ai/               # AI router & processing
│   ├── auth/             # Authentication & authorization
│   └── config/           # Configuration management
├── pkg/                   # Public packages
│   ├── models/           # Data models
│   ├── utils/            # Utility functions
│   └── errors/           # Error types
├── web/                   # React frontend
│   ├── src/              # Source files
│   ├── public/           # Static assets
│   └── dist/             # Production build
├── docs/                  # Documentation
│   ├── architecture.md   # System architecture
│   ├── getting-started.md
│   └── api.md            # API reference
├── deployments/           # Deployment configurations
│   ├── k8s-deployment.yaml
│   └── systemd/
├── .github/              # GitHub Actions workflows
│   └── workflows/
│       ├── ci.yml        # CI pipeline
│       ├── security-scan.yml
│       └── release.yml
└── scripts/              # Utility scripts
```

---

## 📚 Documentation

- [Getting Started Guide](docs/getting-started.md)
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api.md)
- [Security Best Practices](SECURITY.md)
- [Contributing Guide](CONTRIBUTING.md)
- [Deployment Guide](DEPLOYMENT.md)
- [Building & Installation](QUICKSTART_DOCKER.md)

---

## 🔒 Security

See [SECURITY.md](SECURITY.md) for:
- Security policy and incident response
- BSI C5 compliance information
- Vulnerability disclosure process
- Security headers and configurations
- Audit logging configuration

### Security Features

- **BSI C5 Compliance**: Follows German federal standards for IT security
- **Container Scanning**: Trivy integration for vulnerability detection
- **Secret Detection**: TruffleHog for detecting hardcoded secrets
- **Dependency Scanning**: npm audit, govulncheck for CVE detection
- **Static Analysis**: golangci-lint, gosec, codeQL
- **Audit Logging**: Comprehensive request/response logging
- **Rate Limiting**: API rate limiting and protection
- **CORS Protection**: Configurable CORS policies
- **HTTPS Enforcement**: TLS 1.3 with modern ciphers

---

## 🧪 Testing

```bash
# Run all tests
go test -v -race -timeout 10m -coverprofile=coverage.out ./...

# Run frontend tests
cd web && npm test

# Run E2E tests
./test-e2e.sh

# Generate coverage report
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## 📊 Monitoring & Observability

Axiom includes comprehensive observability features:

- **Health Check**: `/health` endpoint for liveness/readiness
- **Metrics**: Prometheus metrics at `/metrics`
- **Tracing**: OpenTelemetry integration ready
- **Logs**: Structured logging to JSON
- **Audit**: Request/response audit logs
- **Alerts**: Webhook notifications for critical events

---

## 🌐 API Reference

### Health Check

```bash
curl http://localhost:8080/health
```

### Service Catalog

```bash
curl http://localhost:8080/api/v1/catalog/services
```

### AI Assistant

```bash
curl -X POST http://localhost:8080/api/v1/ai/query \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What services are available?"
  }'
```

### Authentication

```bash
curl http://localhost:8080/api/v1/auth/login
```

---

## 🛠️ MCP Servers

### Built-in MCP Servers

- **GitHub MCP**: Repository management, pull request automation
- **Kubernetes MCP**: Cluster management, deployment automation
- **Terraform MCP**: Infrastructure as code management
- **AWS MCP**: AWS service management and automation

### Adding Custom MCP Servers

1. Create MCP server directory: `mcp-servers/custom-server/`
2. Implement JSON-RPC 2.0 spec
3. Register with `internal/mcp/registry.go`
4. Add to `.env`: `MCP_SERVERS=...,custom-server`

---

## 🚀 Deployment

### Docker Compose

```bash
docker-compose up -d
```

### Kubernetes

```bash
kubectl apply -k deployments/k8s/
```

### Systemd Service

```bash
sudo ./scripts/install-systemd.sh
sudo systemctl enable axiom-server
sudo systemctl start axiom-server
```

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development workflow
- Pull request process
- Code style guidelines
- Testing requirements
- Documentation standards

### Getting Started

```bash
git clone https://github.com/axiom-idp/axiom.git
cd axiom
git checkout -b feature/my-feature
make build
make dev
```

### Code of Conduct

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) before contributing.

---

## 📝 License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

**Copyright © 2026 Nishant Ravi <nishaero@gmail.com>**

---

## 📞 Support

- 📖 [Documentation](./docs)
- 💬 [GitHub Discussions](https://github.com/axiom-idp/axiom/discussions)
- 🐛 [Issue Tracker](https://github.com/axiom-idp/axiom/issues)
- 🔐 [Security Issues](SECURITY.md)
- 📧 Email: nishaero@gmail.com

---

## 🗺️ Roadmap

- [x] Phase 1: Project Foundation ✅
- [x] Phase 2: Backend Implementation ✅
- [x] Phase 3: Frontend Implementation ✅
- [x] Phase 4: Security Layer ✅
- [x] Phase 5: CI/CD Workflows ✅
- [x] Phase 6: Deployment Setup ✅
- [x] Phase 7: MCP Servers ✅
- [x] Phase 8: Testing & E2E ✅
- [ ] Phase 9: Performance Optimization
- [ ] Phase 10: Multi-Tenant Support
- [ ] Phase 11: GraphQL API
- [ ] Phase 12: Advanced AI Features

---

<div align="center">
  <strong>Made with ❤️ by the Axiom Team</strong>
  <br><br>
  <a href="https://github.com/axiom-idp/axiom">github.com/axiom-idp/axiom</a>
</div>
