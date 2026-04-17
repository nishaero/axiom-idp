# Axiom IDP - Project Status

Updated: 2026-04-17
Branch: `main`

## Executive Summary

Axiom IDP is in release-readiness hardening on `main`.

The repository is aligned with the intended product direction:

- AI-assisted release readiness and operational guidance
- GitHub-native SDLC and delivery governance
- GitOps-capable deployment flows
- BSI C5-aligned security baseline
- production-oriented Docker and Kubernetes deployment posture

## Validated Status

Local validation completed:

- `go test ./...`
- `go vet ./...`
- `cd web && npm run lint`
- `cd web && npm test -- --run`
- `cd web && npm run build`
- `./scripts/validate-docker.sh`
- `./scripts/validate-minikube.sh`

GitHub validation has been completed for the merged delivery changes and is now being tightened on `main` around automated release gating:

- `CI`
- `Code Quality Gate`
- `Security Scan`
- `Dependency Review`

## What Is Working

Backend:

- signed token auth and RBAC
- security headers, audit logging, and rate limiting
- `/live`, `/ready`, `/health`, and `/api/v1/platform/status`
- `/metrics` Prometheus-style telemetry export and `/api/v1/platform/observability`
- catalog overview, service analysis, and AI query endpoints using an OpenAI-compatible request path with Ollama as the local provider option
- direct Kubernetes deployment flow
- GitHub-backed Argo CD deployment flow
- Terraform-backed infrastructure request flow through GitOps execution
- async deployment and infrastructure jobs with `/api/v1/jobs` and `/api/v1/jobs/{id}`

Frontend:

- decision-oriented dashboard
- dedicated observability view with endpoint checks and telemetry counters
- service catalog with release-readiness drilldown
- release briefs with exportable evidence packs and next-best-action guidance
- AI assistant with deployment and infrastructure workflow tracking
- settings and governance UX
- live platform status surface backed by the API

GitHub and release automation:

- CI, quality, security, dependency review, image validation, and deploy validation
- issue triage, stale handling, label synchronization
- automatic semver tagging after a fully successful merged-commit workflow set
- GHCR image publication
- signed images, SBOM generation, and provenance attestation in release/publish workflows

## Product Position

Axiom is positioned as an AI-native internal developer platform for:

- release readiness
- delivery intelligence
- compliance-aware change control
- GitHub-centric software delivery

The goal is not to replicate generic portal products. The differentiator is the combination of:

- deterministic platform analysis
- AI-assisted operational decisions
- evidence-aware workflows
- GitOps-compatible delivery control

## Known Boundaries

The repository is close to launch, but these items are still outside the current validated baseline:

- formal BSI C5 certification
- full HA architecture with externalized state for every runtime subsystem
- complete Crossplane execution validation
- durable async job orchestration across replicas
- full observability stack with Prometheus/OpenTelemetry/Grafana
- browser-driven E2E automation for all UI journeys

## Next Recommended Step

Move from merged-main validation to the first automated release:

1. complete the remaining green `main` workflow run set
2. confirm `Auto Tag Release` creates the first semantic version tag
3. verify the generated GitHub release assets, signed GHCR image, SBOM, and attestations
4. deploy with production secrets and backing services
5. complete the post-release launch checklist
