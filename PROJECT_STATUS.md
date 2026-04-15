# Axiom IDP - Project Status

Updated: 2026-04-15
Branch: `feat/production-delivery-baseline`

## Executive Summary

Axiom IDP is now in a working, validated launch-candidate state.

The repository has been brought back into alignment with the intended product direction:

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

GitHub validation on the current branch completed successfully:

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

- CI, quality, security, dependency review, deploy validation
- issue triage, stale handling, label synchronization
- semver-tagged release workflow
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

The branch is ready for controlled launch, but these items are still outside the current validated baseline:

- formal BSI C5 certification
- full HA architecture with externalized state for every runtime subsystem
- complete Crossplane execution validation
- durable async job orchestration across replicas
- full observability stack with Prometheus/OpenTelemetry/Grafana
- browser-driven E2E automation for all UI journeys

## Next Recommended Step

Move from branch validation to release:

1. merge the branch
2. create a semantic version tag such as `v1.0.0`
3. verify signed GHCR artifacts and attestations
4. deploy with production secrets and backing services
5. complete the post-merge launch checklist
