---
title: "SB-710: Hand-off game day with a friendly tenant"
summary: "Rehearse it before anyone needs it."
---

# SB-710: Hand-off game day with a friendly tenant

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.3 Exit paths |
| Stories | US-08.5 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-706](../E08-factory-handoff/SB-706-access-path-enumeration-and-law-17-check.md), [SB-707](../E08-factory-handoff/SB-707-releases-after-hand-off.md), [SB-708](../E08-factory-handoff/SB-708-detach-and-export-package.md) |
| Status | In review |

## Why

Rehearse it before anyone needs it.

## Acceptance criteria

- [ ] A friendly tenant's cell handed off end to end; laws pass; audit chain verifies across the move (rehearsed in `tests/handoff`; the day with a friendly tenant is human)
- [ ] Detach rehearsed on a copy (rehearsed in `tests/handoff`; on the day, by people)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Plan
- [ ] Run (human, with a friendly tenant: [runbook](../../docs/runbooks/handoff-game-day.md))
- [ ] Retro, bugs and runbook fixes (the rehearsal's own retro: BUG-0012, fixed)
- [x] Write or extend the law tests: LAW-17, LAW-18
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.3c): sb-factory(7) gained "Hand-off game day, exactly", the six checks a day must pass. `docs/runbooks/handoff-game-day.md` covers plan, run and retro, with an expected result for every step. `tests/handoff`, now in `make check`, rehearses the whole day. A factory cell in a local account, with audit rows, approvals, an index and a key wrap, is handed to a tenant's GCP project (the recorded fake, with the bootstrap's bindings and two factory paths that must go). It goes through the real controller, wrap inventory, backup and restore, index rebuild, access check and registry, and is then detached. It checks LAW-17 (only the retained `platform-kill` binding is left), LAW-18 (only the tenant's KMS key unwraps), the audit chain moved intact with nine custody rows, the custody record, and after detach, no paths and a tombstone. The rehearsal found a real bug, BUG-0012. The controller matched retained grants against path names, so cutover revoked the retained `platform-kill` binding. Its Access port now returns paths with the grant each serves, and the hand-off law tests and unit tests follow.

**Left:** the game day itself with a friendly tenant (human).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
