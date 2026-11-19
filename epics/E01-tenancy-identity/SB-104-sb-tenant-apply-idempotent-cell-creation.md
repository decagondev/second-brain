---
title: "SB-104: `sb tenant apply` idempotent cell creation"
summary: "Applying twice must be the same as applying once."
---

# SB-104: `sb tenant apply` idempotent cell creation

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.2 Cell |
| Stories | US-01.1 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-102](../E01-tenancy-identity/SB-102-sb-tenant-apply-plan-dry-run.md), [SB-103](../E01-tenancy-identity/SB-103-cell-template-infrastructure-as-code.md) |
| Status | In review |

## Why

Applying twice must be the same as applying once.

## Acceptance criteria

- [ ] New cell in under 30 minutes (real cloud only; the local adapter takes milliseconds)
- [x] Second apply with no changes reports `no changes` and touches nothing
- [x] Apply is resumable after a crash at any step

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Apply engine with step journal
- [x] Crash-at-each-step test
- [ ] Timing measurement in CI staging (needs a cloud adapter and a staging account)
- [ ] Audit row per applied step (the journal records every step now; it feeds the audit chain when SB-401 lands)
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built on the local control plane (ADR-0018). `internal/core/apply` runs the plan one change at a time. A new cell is stamped first and recorded in the registry only after all five resources exist. Every step is journalled (`start`/`done`) through the new `ports.Journal`, with ids and names only, and the registry gets `ports.RegistryWriter`. A no-change apply writes nothing. A registry that can't answer is an error, never "no cell". The crash test fails each write in turn (41 crash points across a create and a change), reruns, and gets the same registry record, no duplicate resources, a journal where every `done` follows its `start`, and an empty follow-up plan. `sb tenant apply FILE` prints each change and a summary. The smoke test runs create, no changes, then pilot to full through the real binary. LAW-16 gains an apply-side law test.

**Left:** the 30-minute timing and the staging measurement need a cloud adapter. Audit rows per step wait for the audit chain (SB-401); the journal is already their source. Also left: review by the user, and the first green CI run.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
