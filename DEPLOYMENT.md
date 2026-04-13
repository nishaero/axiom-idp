# Axiom IDP - Docker Deployment

## Quick Start

```bash
# Build and start the container
docker compose up -d

# Check status
docker ps | grep axiom

# View logs
docker logs -f axiom-idp-axiom-1

# Stop
docker compose down

# Stop and remove volumes
docker compose down -v
```

## Access Points

| Service | URL | Status |
|---------|-----|--------|
| Frontend | http://localhost:8080 | Running |
| Health Check | http://localhost:8080/health | Running |
| API | http://localhost:8080/api/v1 | Running |

## Kubernetes Deployment

```bash
# Apply Kubernetes manifests
kubectl apply -f deployments/k8s-deployment.yaml

# Check pods
kubectl get pods -n axiom

# Port forward to access
kubectl port-forward -n axiom svc/axiom-server 8080:80
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| AXIOM_PORT | Server port | 8080 |
| AXIOM_ENV | Environment (dev/prod) | development |
| AXIOM_LOG_LEVEL | Log level (debug/info/warn/error) | debug |
| AXIOM_DB_DRIVER | Database driver (sqlite3/postgres) | sqlite3 |
| AXIOM_DB_URL | Database connection string | ./data/axiom.db |
| AXIOM_SESSION_SECRET | Session secret (change in production) | dev-secret-change-in-production |

## Volumes

- `axiom-data` - Persistent storage for SQLite database and application data

## Network

- `axiom-network` - Docker bridge network for internal communication

## Production Deployment

For production, you should:

1. Change `AXIOM_SESSION_SECRET` to a strong random value
2. Use PostgreSQL instead of SQLite
3. Configure TLS/HTTPS
4. Set up proper RBAC and authentication
5. Use resource limits
6. Enable audit logging

See `docs/` for more information.
