# Axiom IDP - Docker Deployment

## Quick Start

```bash
# Build and start the container
docker compose up -d --build

# Check status
docker ps | grep axiom

# View logs
docker compose logs -f

# Stop
docker compose down

# Stop and remove volumes
docker compose down -v
```

## AI Runtime Modes

- Local fallback mode: `AXIOM_AI_BACKEND=local` uses the embedded assistant and does not require Ollama.
- Ollama mode: `AXIOM_AI_BACKEND=ollama` sends requests to a reachable Ollama endpoint and falls back to local mode if the upstream call fails.

Example for a local Ollama service on the host:

```bash
AXIOM_AI_BACKEND=ollama \
AXIOM_AI_BASE_URL=http://host.docker.internal:11434 \
AXIOM_AI_MODEL=qwen3.5:9b \
docker compose up -d --build
```

## Access Points

| Service | URL | Status |
|---------|-----|--------|
| Frontend | http://localhost:8080 | Running |
| Health Check | http://localhost:8080/health | Running |
| API | http://localhost:8080/api/v1 | Running |

## Kubernetes Deployment

The Kubernetes manifest defaults to local fallback mode. Patch the `AXIOM_AI_*` settings before rollout if you want to use Ollama instead.

```bash
# Apply Kubernetes manifests
kubectl apply -f deployments/k8s-deployment.yaml

# Check pods
kubectl get pods -n axiom

# Port forward to access
kubectl port-forward -n axiom svc/axiom-server 8080:80
```

## Validation Helpers

```bash
# Docker Compose validation
make verify-docker

# Minikube validation
make verify-minikube

# Generic app probe
./verify-app.sh
```

## GitHub Governance Bootstrap

After creating or importing the repository, run:

```bash
./scripts/bootstrap-github-governance.sh
```

This configures:
- managed labels from `.github/labels.json`
- merge settings for squash-only history
- auto-delete branch on merge
- auto-merge support
- signoff requirement on web commits
- branch protection on the repository default branch

The bootstrap script assumes the following checks exist in GitHub Actions:
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

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| AXIOM_HOST | Bind address | 0.0.0.0 |
| AXIOM_PORT | Server port | 8081 |
| AXIOM_ENV | Environment (development/production) | development |
| AXIOM_LOG_LEVEL | Log level (debug/info/warn/error) | info |
| AXIOM_DB_DRIVER | Database driver | sqlite3 |
| AXIOM_DB_URL | Database connection string | file:axiom.db |
| AXIOM_SESSION_SECRET | Session secret (change in production) | REPLACE_WITH_A_LONG_RANDOM_SECRET |
| AXIOM_AI_BACKEND | AI mode (`local` or `ollama`) | local |
| AXIOM_AI_BASE_URL | Ollama base URL | http://127.0.0.1:11434 |
| AXIOM_AI_MODEL | Ollama model name | qwen3.5:9b |
| AXIOM_AI_TIMEOUT | AI request timeout | 90s |
| AXIOM_AI_MAX_TOKENS | Ollama generation limit | 768 |

## Volumes

- `axiom-data` - Persistent storage for application data

## Network

- `axiom-network` - Docker bridge network for internal communication

## Production Deployment

For production, you should:

1. Change `AXIOM_SESSION_SECRET` to a strong random value
2. Decide whether the deployment uses local fallback mode or Ollama mode and set the matching `AXIOM_AI_*` variables
3. Configure TLS/HTTPS
4. Set up proper RBAC and authentication
5. Use resource limits
6. Enable audit logging
7. Pull images from GitHub Container Registry (`ghcr.io/axiom-idp/axiom`)

See `docs/` for more information.
