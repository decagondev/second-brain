---
title: "ADR-0007: Go for services and CLI; Python for evals; TypeScript for UI"
summary: "Go for services and CLI; Python for evals; TypeScript for UI."
---

# ADR-0007: Go for services and CLI; Python for evals; TypeScript for UI

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

The platform needs a fast, boring, statically typed language for services and a single-binary CLI that portfolio IT can install anywhere.

## Decision

Go for all services, the policy host and the `sb` CLI. Python for evals and data tooling, where the ecosystem is strongest. TypeScript for the web console and chat UI, server-rendered with minimal client JS. One language per layer; no others without an ADR.

## Alternatives rejected

| Option | Why not |
|---|---|
| TypeScript everywhere | Weaker static guarantees and heavier runtime for the core |
| Rust | Stronger guarantees, but slower team ramp-up and the safety-critical logic is in policy and proofs, not memory safety |
| C++ | No benefit for a network service of this shape; higher defect risk |

## Consequences

- Single static binary for `sb`
- Property testing via `pgregory.net/rapid`; lint via `golangci-lint` with `depguard` for the dependency rule
