---
title: "ADR-0008: Write policy in Cedar"
summary: "Write policy in Cedar."
---

# ADR-0008: Write policy in Cedar

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Policy must be readable by company operators, analysable by tools, and fast. Hand-written authorisation code is none of those.

## Decision

Platform and tenant policies are written in Cedar, validated against a schema in CI, and evaluated by the policy engine. Cedar's default-deny, forbid-overrides-permit semantics match P1. Source ACLs stay with the source (ADR-0002); Cedar decides everything else.

## Alternatives rejected

| Option | Why not |
|---|---|
| Authorisation in Go code | Not reviewable by operators; not analysable |
| OPA/Rego | Powerful but general-purpose; harder to read and to analyse for our narrow use |

## Consequences

- Policies can be diffed and reviewed like code (SB-112)
- A Go binding is needed; isolated behind the `PolicyEvaluator` port
