# Axiom IDP - AI-Native Internal Developer Platform

[![Build Status](https://github.com/nishaero/axiom-idp/actions/workflows/ci.yml/badge.svg)](https://github.com/nishaero/axiom-idp/actions)
[![Security Scanning](https://github.com/nishaero/axiom-idp/actions/workflows/security-scan.yml/badge.svg)](https://github.com/nishaero/axiom-idp/actions)
[![License](https://img.shields.io/badge/license-PolyForm%20Noncommercial%201.0.0-orange)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/nishaero/axiom-idp)](https://goreportcard.com/report/github.com/nishaero/axiom-idp)
[![Code Coverage](https://codecov.io/github/nishaero/axiom-idp/graph/badge.svg)](https://codecov.io/github/nishaero/axiom-idp)

<div align="center">
  <img src="docs/assets/axiom-logo.svg" alt="Axiom IDP Logo" width="200" style="border-radius: 12px;" />
  <br>
  <strong>Stateless, MCP-Native Internal Developer Platform with AI-First Design</strong>
  <br>
  <a href="https://github.com/nishaero/axiom-idp/releases/latest">Download Latest Release</a>
  <br><br>
  <a href="SECURITY.md"><img src="https://img.shields.io/badge/BSI%20C5-aligned%20baseline-green" alt="BSI C5 aligned baseline"></a>
  <a href="https://polyformproject.org/licenses/noncommercial/1.0.0/"><img src="https://img.shields.io/badge/License-PolyForm%20Noncommercial%201.0.0-orange" alt="License"></a>
</div>

---

## 🎯 Overview

Axiom is an AI-native internal developer platform focused on release readiness, GitOps delivery, and compliance-aware operations. It combines deterministic backend analysis with local or Ollama-backed AI guidance, GitHub-native SDLC automation, and deployment flows that can act directly on Kubernetes or through GitOps with Argo CD.

### AI Runtime Modes

- `AXIOM_AI_BACKEND=local` is the default and works without Ollama.
- `AXIOM_AI_BACKEND=ollama` sends prompts to a reachable Ollama server.
- If Ollama fails or returns an empty response, the request falls back to local mode and the API marks the backend as `local-fallback`.

### ✨ Key Features

- **AI-Native Architecture**: Deterministic platform analysis with local or Ollama-backed AI guidance
- **GitOps Delivery**: AI can trigger direct Kubernetes deploys or GitHub-backed Argo CD delivery flows
- **Infrastructure Workflows**: Terraform-backed infrastructure requests are supported through GitOps execution
- **Optional MCP Integration Plane**: MCP remains available for pluggable AI-facing integrations without owning the core runtime
- **RBAC & OAuth2/OIDC**: Enterprise-grade security with fine-grained access control
- **Professional UI**: Modern React + TypeScript + Tailwind CSS frontend
- **Production-Ready Delivery Paths**: Docker, Kubernetes, GitHub Actions, GHCR, and semver-tagged release automation
- **Security First**: BSI C5-aligned baseline, signed images, SBOM generation, provenance attestation, vulnerability scanning
- **Operational Visibility**: Audit trails, readiness/liveness endpoints, and backend-fed platform status in the UI

---

## 📥 Quick Start

### Prerequisites

- Go 1.24+
- Node.js 24+ and npm
- Docker and Docker Compose v2
- Optional: Ollama reachable from the deployment target if you want the AI-backed mode

### Installation

#### Option 1: From GitHub Release (Recommended)

```bash
# Download latest release
curl -L https://github.com/nishaero/axiom-idp/releases/latest/download/axiom-linux-amd64 -o axiom
chmod +x axiom
sudo mv axiom /usr/local/bin/
```

#### Option 2: Build from Source

```bash
git clone https://github.com/nishaero/axiom-idp.git
cd axiom-idp
make build
./bin/axiom-server
```

### Configuration

Create a `.env` file in the root directory:

```env
AXIOM_HOST=0.0.0.0
AXIOM_PORT=8081
AXIOM_ENV=production
AXIOM_LOG_LEVEL=info
AXIOM_SESSION_SECRET=replace-with-a-long-random-secret
AXIOM_AI_BACKEND=local
# AXIOM_AI_BACKEND=ollama
# AXIOM_AI_BASE_URL=http://host.docker.internal:11434
# AXIOM_AI_MODEL=qwen3.5:9b
```

For Ollama-backed runs, start Ollama separately and pull the target model before starting Axiom:

```bash
ollama pull qwen3.5:9b
ollama serve
```

### Running

```bash
# Start the server with local fallback mode
AXIOM_AI_BACKEND=local ./bin/axiom-server

# Or with Docker Compose
docker compose up -d --build

# Or with Ollama on your machine
AXIOM_AI_BACKEND=ollama AXIOM_AI_BASE_URL=http://host.docker.internal:11434 docker compose up -d --build
```

Visit `http://localhost:8080` in your browser.

### Runtime Status Endpoints

- `/live` returns process liveness
- `/ready` returns runtime readiness
- `/health` returns platform health summary
- `/api/v1/platform/status` returns backend-fed operational status used by the dashboard

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
docker compose up -d --build
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
- [Platform Readiness Assessment](docs/platform-readiness-assessment.md)
- [Market Research and Differentiation](docs/market-research.md)
- [Security Best Practices](SECURITY.md)
- [Contributing Guide](CONTRIBUTING.md)
- [Deployment Guide](DEPLOYMENT.md)
- [Building & Installation](QUICKSTART_DOCKER.md)

---

## 🚀 Release And Supply Chain

Axiom uses GitHub-native release automation:

- Pull requests must pass `CI`, `Code Quality Gate`, `Security Scan`, and `Dependency Review`
- release workflow triggers only on semantic version tags like `v1.2.3`
- images are published to `ghcr.io`
- release and validation images are signed with keyless Sigstore cosign
- SPDX SBOMs are generated during publish flows
- build provenance is attested through GitHub artifact attestations

This keeps the SDLC, artifact publication, and verification chain in GitHub instead of splitting release trust across multiple systems.

---

## 🔒 Security

See [SECURITY.md](SECURITY.md) for:
- Security policy and incident response
- BSI C5-aligned baseline controls and deployment requirements
- Vulnerability disclosure process
- Security headers and configurations
- Audit logging configuration

### Security Features

- **BSI C5-aligned baseline**: Hardened container runtime, least-privilege defaults, audit logging, and deployment guidance
- **Container Scanning**: Trivy integration for vulnerability detection
- **Secret Detection**: TruffleHog for detecting hardcoded secrets
- **Dependency Scanning**: `govulncheck` and `npm audit` for CVE detection
- **Static Analysis**: `gosec` and GitHub code scanning
- **Audit Logging**: Comprehensive request/response logging
- **Rate Limiting**: API rate limiting and protection
- **CORS Protection**: Configurable CORS policies
- **HTTPS Enforcement**: TLS 1.3 with modern ciphers

## 🤖 GitHub Governance

The repository is wired for GitHub-native lifecycle automation:
- `labels.json` is the source of truth for managed labels
- issue forms pre-label bug reports and feature requests with `needs-triage`
- PRs are labeled automatically by touched area and marked `needs-review`
- stale issues and PRs are marked with a dedicated `stale` label before closing
- the bootstrap script applies repository settings and branch protection on the default branch

Branch protection assumes these GitHub check names stay stable:
- `Backend Tests`
- `Frontend Tests`
- `Build Container Image`
- `Docker Compose Smoke Test`
- `Container Vulnerability Scan`
- `Go Security Analysis`
- `Secret Detection`
- `Infrastructure as Code Security`
- `Dependency Vulnerability Check`
- `License Compliance Check`

Those checks are enforced before merge when the bootstrap script is used.

---

## 🧪 Testing

```bash
# Run all tests
go test -v -race -timeout 10m -coverprofile=coverage.out ./...

# Run frontend tests
cd web && npm test

# Run E2E tests
./test-e2e.sh

# Validate Docker / Minikube deployments
make verify-docker
make verify-minikube
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
docker compose up -d --build
```

### Kubernetes

```bash
kubectl apply -f deployments/k8s-deployment.yaml
```

### GitHub Container Registry

Release images are published to `ghcr.io/axiom-idp/axiom` and the GitHub release pipeline pushes hardened container builds there.

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

PolyForm Noncommercial 1.0.0 - See [LICENSE](LICENSE) for details. Commercial use requires a separate license from the copyright holder.

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
