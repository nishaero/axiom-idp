# Axiom IDP API Reference

This page documents the release-facing HTTP surface implemented in the current repository state.

## Authentication

- `/health`, `/live`, `/ready`, and `/metrics` are unauthenticated.
- `/api/v1/*` endpoints require a signed session token and the relevant RBAC scope.
- `POST /api/v1/auth/login` creates a session token.
- `POST /api/v1/auth/logout` invalidates the session.

## Platform And Status

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/platform/status` | Backend-fed platform status used by the dashboard |
| `GET` | `/api/v1/platform/observability` | Live observability snapshot with telemetry, health checks, and scrape hints |
| `GET` | `/api/v1/jobs` | Lists async deployment and infrastructure jobs |
| `GET` | `/api/v1/jobs/{id}` | Returns the current state of a specific async job |

## Catalog

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/catalog/services` | Returns the catalog view |
| `GET` | `/api/v1/catalog/search` | Searches catalog entries by query |
| `GET` | `/api/v1/catalog/overview` | Returns portfolio summary counts |
| `GET` | `/api/v1/catalog/services/{id}` | Returns a service insight view |
| `GET` | `/api/v1/catalog/services/{id}/analysis` | Alias for the service insight view |

## AI And Execution

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/ai/query` | Accepts a natural-language request with a `query` field and returns deterministic analysis with optional OpenAI-compatible AI guidance |
| `POST` | `/api/v1/deployments/applications` | Starts an application deployment request and returns a job record |
| `GET` | `/api/v1/deployments/applications/{namespace}/{name}` | Returns deployment status for a specific application |

The AI request path speaks an OpenAI-compatible chat-completions contract when `AXIOM_AI_BACKEND` is set to `ollama` or `openai`. Local fallback mode remains available when no provider is configured.

## Example Requests

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "demo-user",
    "roles": ["viewer"]
  }'
```

```bash
curl -X POST http://localhost:8080/api/v1/ai/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "query": "summarize release readiness for payments-api"
  }'
```

## Notes

- Deployment and infrastructure actions are queued through the running process. The job queue is currently process-local.
- Runtime state for audit history and rate limiting can be backed by SQLite for local runs or PostgreSQL for shared deployments.
