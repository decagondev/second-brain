---
title: "ADR-0011: No Kubernetes in v1; use a managed container runtime"
summary: "No Kubernetes in v1; use a managed container runtime."
---

# ADR-0011: No Kubernetes in v1; use a managed container runtime

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Every cell is a handful of stateless services plus managed data stores. Kubernetes would add a platform to operate per region for no product benefit yet.

## Decision

Run services on a managed container runtime with per-cell service accounts. Revisit when tenant count or workload shape demands it, with an ADR.

## Alternatives rejected

| Option | Why not |
|---|---|
| Kubernetes | Operational load without a matching need in v1 |

## Consequences

- Less to run and secure
- Some runtime features (sidecars, custom schedulers) aren't available; we design without them
