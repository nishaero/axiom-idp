# Platform Readiness Assessment

Updated: 2026-04-15

## Current Launch State

Axiom IDP is in launch-candidate state on the current branch.

Validated on the current revision:

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

Recent observability additions on this branch:

- Prometheus-style `/metrics` endpoint with HTTP, AI, deployment, audit, and rate-limit telemetry
- backend-fed `/api/v1/platform/observability` snapshot for the UI
- scrape annotations on the Kubernetes Service and Pod template for local/minikube Prometheus discovery
- dedicated observability page in the frontend with endpoint health and telemetry cards

## What Is Production-Ready

- Signed session tokens, RBAC, security headers, rate limiting, and audit middleware
- Local and Ollama-backed AI runtime with deterministic fallback behavior
- GitHub-native SDLC automation with required quality and security gates
- GHCR image publication, semver-tagged release flow, image signing, SBOM generation, and provenance attestation
- Docker and Kubernetes deployment paths with explicit `/live`, `/ready`, and `/health` endpoints
- Backend-fed platform status surfaced in the UI through `/api/v1/platform/status`
- Prometheus-compatible metrics exposure through `/metrics`
- dedicated observability snapshot and UI page for endpoint health, scrape hints, and telemetry counters
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

This branch is validated and deployable, but a few items are still beyond the current launch baseline:

- formal BSI C5 certification still requires organizational controls, audit evidence, and external review
- Crossplane execution remains controller-dependent and is surfaced explicitly as staged until reconciliation is available
- true HA at production scale still needs externalized state for database, audit/event storage, and rate limiting
- telemetry counters are in-process and suitable for local or single-instance deployments, but a shared metrics backend is still recommended for multi-replica durability
- a full Prometheus/OpenTelemetry/Grafana stack is still a recommended next step for production-grade retention and alerting
- browser-driven E2E coverage for every UI path is not yet part of the automated suite

## Recommendation

The repository is ready to move from implementation to controlled launch activity:

1. merge the validated branch
2. cut a semantic version tag
3. verify the published GHCR image signature, SBOM, and provenance
4. deploy to the target environment with a real session secret and production backing services
5. complete the final organization-specific readiness checklist for secrets, HA, observability, and compliance evidence
