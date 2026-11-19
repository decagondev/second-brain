---
title: "ADR-0009: The manual is the spec, and CI enforces it"
summary: "The manual is the spec, and CI enforces it."
---

# ADR-0009: The manual is the spec, and CI enforces it

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Docs written after code describe what was built, not what was meant. Users, operators and LLM agents all read the docs first.

## Decision

Every behaviour change starts as a `manual/` change. CI fails a PR that touches `cmd/` or `internal/` without touching `manual/`, unless it's labelled `refactor-no-behaviour`. Manual examples tagged `run` are executed in CI.

## Alternatives rejected

| Option | Why not |
|---|---|
| Docs as a release task | Always late, always wrong |

## Consequences

- Every ticket's first subtask is the man page
- Small friction on every PR; the label covers real refactors
