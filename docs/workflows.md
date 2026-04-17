# Workflow Guide

Updated: 2026-04-17

## Purpose

This repository keeps the SDLC, release, and governance path in GitHub.

## Validation Workflows

- `CI` runs backend tests, frontend tests, image build checks, and smoke validation.
- `Code Quality Gate` runs workflow linting, `go vet`, frontend type-checking, frontend linting, and optional Sonar analysis.
- `Security Scan` runs container scanning, Go security analysis, secret detection, IaC scanning, dependency vulnerability checks, and license reporting.
- `Dependency Review` checks dependency changes on pull requests.

## Release Workflows

- `Image Publish Validation` builds and validates the container image path used for releases.
- `Deploy Validation` validates the GitOps and Minikube deployment path for merged `main` commits.
- `Auto Tag Release` watches the merged-commit workflow set on `main` and creates the next patch semver tag when `CI`, `Code Quality Gate`, `Security Scan`, `Image Publish Validation`, and `Deploy Validation` all succeed for the same `main` SHA.
- `Release` runs from semantic version tags and publishes signed artifacts, signed GHCR images, SBOMs, and provenance attestations.

## Governance Workflows

- `Triage` labels pull requests automatically based on changed areas.
- `Sync Labels` keeps GitHub labels aligned with `.github/labels.json`.
- `Stale` marks and closes inactive issues and pull requests under the configured policy.

## Branch Protection Assumptions

The protected default branch is expected to require:

- linear history
- code owner review
- last-push approval
- the repository’s required status checks

If branch protection changes, update both the GitHub repository settings and `scripts/bootstrap-github-governance.sh`.
