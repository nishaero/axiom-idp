# Axiom IDP Market Research and Differentiation

## Purpose

This document records the market signals used to shape Axiom IDP. It is a gap analysis, not a template for copying competitor implementations. Axiom must stay original in architecture, workflows, language, and user experience.

## Market Signals

As of April 14, 2026, the main Internal Developer Platform and developer portal vendors already emphasize:

- Software catalogs and service inventory
- Scorecards and standards tracking
- Self-service workflows
- CI/CD and cloud integration
- AI-assisted discovery or operational context

Representative sources:

- Backstage overview and software catalog positioning:
  - https://backstage.io/
- Port scorecards and standards tracking:
  - https://docs.port.io/guides/all/measure-standards
- Cortex scorecards:
  - https://docs.cortex.io/scorecards/create
- Humanitec platform orchestrator:
  - https://humanitec.com/products/platform-orchestrator

## What This Means

“Catalog + scorecards + workflows + AI assistant” is no longer a differentiator by itself. Those capabilities are now baseline expectations in the market.

The opportunity for Axiom is to move from passive portal behavior to active release governance with evidence, reasoning, and risk controls that are useful for regulated environments.

## Axiom-Specific Differentiators

These are the product areas where Axiom should be original and stronger than current mainstream offerings:

### 1. Evidence-Native BSI C5 Control Mapping

Axiom should generate deployment and change evidence as a first-class product capability, not as an afterthought.

Original direction:

- Per-service evidence packs linked to controls, approvals, runtime health, and delivery events
- Exportable audit narratives for BSI C5 reviews
- Control status derived from live operational signals instead of manual spreadsheet collection

Why it matters:

- Competitors widely support scorecards and compliance-like metadata, but the market messaging reviewed does not center on BSI C5 evidence automation as a core workflow. This is an inference from the cited sources, not a direct claim by those vendors.

### 2. AI-Guided Change Risk With Explainability

Axiom should evaluate a proposed deployment or service change and explain the operational risk in concrete terms.

Original direction:

- Pre-release risk scoring
- Blast-radius hints based on ownership, service health, recent changes, and dependency signals
- Plain-language explanations of why a release is ready, risky, or blocked

Why it matters:

- Current vendors market AI context and orchestration, but there is still room for a system that explains release risk in a compliance- and operations-friendly way rather than acting as a generic chat layer.

### 3. GitHub-Native SDLC Control Plane

Axiom should treat GitHub as the primary system of delivery truth while staying deployable in self-hosted and Kubernetes environments.

Original direction:

- GHCR-first artifact publishing
- Release validation tied to pull requests, container builds, and deployment smoke tests
- Runtime evidence correlated with GitHub changesets and approvals

Why it matters:

- Most competitors integrate with GitHub. Axiom can differentiate by making the GitHub SDLC itself part of the operational evidence chain rather than only another integration source.

### 4. Operator-Facing “Decision” UX, Not Just Dashboard UX

Axiom should optimize for the moment a platform engineer decides whether to approve, delay, or investigate a release.

Original direction:

- Decision-oriented summaries
- Required-owner and readiness checks surfaced before deployment
- Focus on “what should I do next” rather than generic observability tiles

Why it matters:

- Many portals are strong at inventory and metadata presentation. Fewer are shaped around a release decision loop with evidence and risk reasoning as the primary interaction.

## Design Guardrails

To keep Axiom unique:

- Do not copy competitor page structure, terminology, or visual hierarchy.
- Do not mirror competitor workflow names.
- Keep Axiom language focused on evidence, readiness, and operational decisions.
- Prefer original data models that connect service health, delivery state, ownership, and compliance evidence.

## Product Recommendation

The strongest near-term positioning for Axiom is:

“An AI-native internal developer platform for release readiness and compliance evidence, built around GitHub delivery flows and BSI C5-aligned operational controls.”

That positioning is narrower than a generic portal, but it is more defensible and more original.
