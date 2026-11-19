---
title: "ADR-0005: Reproducible, signed builds with SBOM; lockfiles regenerated every release"
summary: "Reproducible, signed builds with SBOM; lockfiles regenerated every release."
---

# ADR-0005: Reproducible, signed builds with SBOM; lockfiles regenerated every release

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

To prove which build ran for any action, two builds of the same commit must match, and the deployed artefact must be the gated one.

## Decision

Builds are hermetic and reproducible. Each release is built twice independently and the digests compared. Artefacts are signed and carry an SPDX SBOM. Cells refuse unsigned or unverified images. Lockfiles are regenerated every release so dependencies don't fossilise.

## Alternatives rejected

| Option | Why not |
|---|---|
| Sign without reproducibility | A signature proves who built it, not what's in it |
| Pin dependencies indefinitely | CVE exposure accumulates |

## Consequences

- LAW-06 is checkable per cell, hourly
- Build pipeline is stricter and slower; worth it
