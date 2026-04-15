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

- Local fallback mode: `AXIOM_AI_BACKEND=local` uses the embedded assistant and does not require an external provider.
- Ollama mode: `AXIOM_AI_BACKEND=ollama` sends OpenAI-compatible chat-completions requests to a reachable Ollama endpoint.
- OpenAI-compatible mode: `AXIOM_AI_BACKEND=openai` sends the same request shape to another compatible endpoint and requires `AXIOM_AI_API_KEY`.
- If the upstream call fails or returns an empty response, Axiom falls back to local mode and marks the backend as `local-fallback`.

Example for a local provider on the host:

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

The Kubernetes manifest defaults to local fallback mode. Patch the `AXIOM_AI_*` settings before rollout if you want to use an Ollama or other OpenAI-compatible provider.

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
| AXIOM_DB_DRIVER | Runtime state driver (`sqlite3` for local runs, `postgres` for shared HA state) | sqlite3 |
| AXIOM_DB_URL | Runtime state connection string (`file:axiom.db` locally, `postgres://...` in production) | file:axiom.db |
| AXIOM_SESSION_SECRET | Session secret (change in production) | REPLACE_WITH_A_LONG_RANDOM_SECRET |
| AXIOM_AI_BACKEND | AI mode (`local`, `ollama`, or `openai`) | local |
| AXIOM_AI_BASE_URL | OpenAI-compatible base URL | http://127.0.0.1:11434 |
| AXIOM_AI_API_KEY | API key for `openai` mode | empty |
| AXIOM_AI_MODEL | OpenAI-compatible model name | qwen3.5:9b |
| AXIOM_AI_TIMEOUT | AI request timeout | 90s |
| AXIOM_AI_MAX_TOKENS | Generation limit | 768 |

## Volumes

- `axiom-data` - Persistent storage for application data

## Network

- `axiom-network` - Docker bridge network for internal communication

## Production Deployment

For production, you should:

1. Change `AXIOM_SESSION_SECRET` to a strong random value
2. Decide whether the deployment uses local fallback mode, Ollama mode, or another OpenAI-compatible provider and set the matching `AXIOM_AI_*` variables
3. Set `AXIOM_DB_DRIVER=postgres` and point `AXIOM_DB_URL` at a shared PostgreSQL instance for multi-replica audit and rate-limit state
4. Configure TLS/HTTPS
5. Set up proper RBAC and authentication
6. Use resource limits
7. Enable audit logging
8. Pull images from GitHub Container Registry (`ghcr.io/axiom-idp/axiom`)

Note: async deployment and infrastructure jobs are currently handled by the running process, so job state is not shared across replicas yet.

See `docs/` for more information.
