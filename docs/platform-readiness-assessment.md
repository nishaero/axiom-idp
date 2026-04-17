# Platform Readiness Assessment

Updated: 2026-04-17

## Current Launch State

Axiom IDP is in release-readiness hardening on `main`.

Validated locally on the current revision:

- `go test ./...`
- `go vet ./...`
- `cd web && npm run lint`
- `cd web && npm test -- --run`
- `cd web && npm run build`
- `./scripts/validate-docker.sh`
- `./scripts/validate-minikube.sh`
- GitHub Actions:
  - `CI`
  - `Code Quality Gate`
  - `Security Scan`
  - `Dependency Review`
  - `Image Publish Validation`
  - `Deploy Validation`

Recent observability additions on this branch:

- Prometheus-style `/metrics` endpoint with HTTP, AI, deployment, audit, and rate-limit telemetry
- backend-fed `/api/v1/platform/observability` snapshot for the UI
- scrape annotations on the Kubernetes Service and Pod template for local/minikube Prometheus discovery
- dedicated observability page in the frontend with endpoint health and telemetry cards

## What Is Production-Ready

- Signed session tokens, RBAC, security headers, rate limiting, and audit middleware
- Local OpenAI-compatible AI runtime using a provider such as Ollama with deterministic fallback behavior
- GitHub-native SDLC automation with required quality and security gates
- GHCR image publication, automatic semver tagging after merged-commit validation, image signing, SBOM generation, and provenance attestation
- Docker and Kubernetes deployment paths with explicit `/live`, `/ready`, and `/health` endpoints
- Backend-fed platform status surfaced in the UI through `/api/v1/platform/status`
- Prometheus-compatible metrics exposure through `/metrics`
- dedicated observability snapshot and UI page for endpoint health, scrape hints, and telemetry counters
- SQL-backed runtime state for audit history and rate limiting, with PostgreSQL as the shared HA backend and SQLite as the local fallback
- async deployment and infrastructure job tracking through `/api/v1/jobs` and `/api/v1/jobs/{id}`
- Release briefs surfaced in the catalog drilldown with next-best-action guidance and exportable evidence context
- AI-triggered deploy/status flows for:
  - direct Kubernetes deployment
  - GitHub-backed Argo CD deployment
  - Terraform-backed infrastructure requests through GitOps execution
  - Crossplane request bundles with explicit staged/controller-dependent execution metadata

## Competitive Alignment

Axiom is aligned with its intended market position when treated as:

- an AI-assisted release decision platform
- a GitHub-native delivery control plane
- a compliance-aware internal platform with BSI C5-aligned operational guardrails

It is not trying to win as a generic developer portal clone. The strongest differentiators remain:

- release-readiness decisions instead of passive catalog browsing
- evidence-native operational workflows
- GitHub-to-GitOps delivery continuity
- self-hosted AI support for regulated environments

## Remaining Boundaries

The platform is deployable, but a few items are still beyond the current launch baseline:

- formal BSI C5 certification still requires organizational controls, audit evidence, and external review
- the validation harness assumes an OpenAI-compatible request contract, with Ollama used as the local implementation exercised here
- Crossplane execution remains controller-dependent and is surfaced explicitly as staged until reconciliation is available
- true HA at production scale still needs PostgreSQL-backed shared state, plus a shared metrics backend if you want multi-replica durability and retention
- telemetry counters are in-process and suitable for local or single-instance deployments
- async job state is still process-local and should be treated as single-instance state until it is externalized
- a full Prometheus/OpenTelemetry/Grafana stack is still a recommended next step for production-grade retention and alerting
- browser-driven E2E coverage for every UI path is not yet part of the automated suite

## Recommendation

The repository is ready to move from merged-main validation to the first automated release:

1. confirm the full required `main` workflow set is green
2. confirm `Auto Tag Release` creates the first semantic version tag
3. verify the published GHCR image signature, SBOM, and provenance
4. deploy to the target environment with a real session secret, PostgreSQL-backed runtime state, and production backing services
5. complete the final organization-specific readiness checklist for secrets, HA, observability, and compliance evidence
