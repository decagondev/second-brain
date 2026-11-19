---
title: "ADR-0013: Prompts, tools, policies and model version are part of the release"
summary: "Prompts, tools, policies and model version are part of the release."
---

# ADR-0013: Prompts, tools, policies and model version are part of the release

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Changing a prompt changes behaviour as much as changing code. If prompts are edited live, nobody can say what ran.

## Decision

Prompts, tool schemas, platform policies and the model id and parameters are in the release bundle and hashed into the release id. They can't be changed at runtime. A prompt change is a release that passes the eval gate.

## Alternatives rejected

| Option | Why not |
|---|---|
| Live prompt editing in the console | Untestable, unauditable |

## Consequences

- Every audit row's release id identifies the exact prompts and model
- Prompt iteration speed is bounded by the release pipeline; the testing channel and ring 0 keep that fast
