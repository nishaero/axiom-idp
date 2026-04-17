# Release Guide

Updated: 2026-04-17

## Overview

Axiom uses a GitHub-native release chain:

1. pull request validation
2. merged `main` validation
3. automatic semver tagging
4. GitHub release creation
5. GHCR publication with signing, SBOMs, and attestations

## Required Workflow Sequence

Pull requests must pass:

- `CI`
- `Code Quality Gate`
- `Security Scan`
- `Dependency Review`

After merge to `main`, the release gate requires:

- `CI`
- `Code Quality Gate`
- `Security Scan`
- `Image Publish Validation`
- `Deploy Validation`

When that full set succeeds for the same merged commit SHA on `main`, `Auto Tag Release` creates the next patch semver tag automatically. The `Release` workflow then runs from that tag.

## Published Artifacts

Each release publishes:

- server tarballs such as `axiom-server-linux-amd64.tar.gz`
- matching `*.sig` and `*.pem` files for cosign blob verification
- a multi-architecture GHCR image at `ghcr.io/nishaero/axiom-idp`
- an SPDX SBOM for the published image
- GitHub provenance attestations for the image build

## Verify A Release Tarball

```bash
cosign verify-blob axiom-server-linux-amd64.tar.gz \
  --signature axiom-server-linux-amd64.tar.gz.sig \
  --certificate axiom-server-linux-amd64.tar.gz.pem
```

## Verify A Release Image

```bash
cosign verify ghcr.io/nishaero/axiom-idp:v1.0.0 \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp '^https://github.com/nishaero/axiom-idp/.github/workflows/release.yml@refs/tags/v1.0.0$'
```

```bash
gh attestation verify oci://ghcr.io/nishaero/axiom-idp:v1.0.0 --repo nishaero/axiom-idp
```

## Rollback

Rollback is tag-based:

1. select the previous known-good release tag
2. redeploy the matching image digest or release image tag
3. confirm `/ready`, `/health`, `/api/v1/platform/status`, and `/api/v1/platform/observability`
4. review audit logs and deployment job records before resuming forward changes
