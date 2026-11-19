---
title: "ADR-0003: An action ran only if the target returned a receipt"
summary: "An action ran only if the target returned a receipt."
---

# ADR-0003: An action ran only if the target returned a receipt

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Models can and do claim to have done things they didn't do. Users then act on a false 'done'.

## Decision

An action's state becomes `Ran` only when the connector's `act` call returns a target-issued receipt id. Anything else (error, timeout, success without a receipt) is `DidNotRun`, shown plainly. The model's claim is ignored.

## Alternatives rejected

| Option | Why not |
|---|---|
| Trust HTTP 2xx | Some APIs return 2xx for queued or partially applied work |
| Trust the model's tool-result summary | The model is userspace (P5) |

## Consequences

- Users never see a false success (LAW-05)
- Connectors must implement idempotency and return receipt ids; some vendors make this awkward and need a read-back
