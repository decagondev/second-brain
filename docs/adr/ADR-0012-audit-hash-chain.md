---
title: "ADR-0012: Audit is an append-only hash chain in the cell database, anchored hourly to WORM storage"
summary: "Audit is an append-only hash chain in the cell database, anchored hourly to WORM storage."
---

# ADR-0012: Audit is an append-only hash chain in the cell database, anchored hourly to WORM storage

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Auditors need tamper evidence, and the platform needs audit writes on the hot path with transactional guarantees.

## Decision

Audit rows live in their own schema in the cell database. The audit role can only INSERT and SELECT. Each row contains the previous row's hash. Every hour the latest hash is signed and written to WORM object storage. `sb audit verify` checks the chain against the anchors.

## Alternatives rejected

| Option | Why not |
|---|---|
| External ledger service | Adds a network hop to every tool call and a new failure mode |
| Plain append-only table | Edits by a privileged user would go unnoticed |

## Consequences

- Write-ahead audit is a local transaction (LAW-04)
- Tampering is detectable, not impossible; the anchors bound how much can be rewritten undetected to the last hour
