---
title: "SB-1507: The CLI commands that need a running cell"
summary: "`sb audit tail`, `sb audit export`, `sb eval diff`, `sb factory handoff` and `sb factory detach` answer `ENOTIMPL` today."
---

# SB-1507: The CLI commands that need a running cell

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.4 |
| Stories | US-15.1 |
| Manual | [`sb(1)`](../../manual/sb.1.md), [`sb-factory(1)`](../../manual/sb-factory.1.md), [`sb-audit(1)`](../../manual/sb-audit.1.md) |
| Laws | LAW-11, LAW-17, LAW-18 |
| Depends on | [SB-1506](SB-1506-apply-to-a-running-cell.md) |
| Status | Done |

## Why

`sb audit tail`, `sb audit export`, `sb eval diff`, `sb factory handoff` and `sb factory detach` answer `ENOTIMPL` today.

## Acceptance criteria

- [x] Each command works against a local cell and leaves `planned`
- [x] Hand-off and detach run their game-day rehearsals against a running cell

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Commands
- [x] Rehearsals
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.4): nothing in `sb` answers `ENOTIMPL` now. The cell has three more control-plane routes (sb-api(3)): audit rows, the signed audit export for a time window (anchored at both ends, so it verifies alone) and the detach package. `sb audit verify --tenant T` (or `--cell URL`) checks a running cell's whole chain with the same code as `--export`; `tail` prints and follows with kind, actor and space filters; `export` writes a window and verifies it first. `sb eval diff` compares two releases' stored results against `evals/gate.yaml` (gate misses, score drops past `drop_max`, newly failing cases, missing suites), and `sb eval gate` is routed too. `sb factory handoff plan|show|approve|execute|abort` runs the hand-off controller on the local control plane with a file store, so a hand-off resumes across invocations, and a `tenant-account://local/ID` target that stamps resources in a second local account, re-wraps the key, moves and verifies the running cell's audit chain, and records the cutover. `sb factory access` checks a handed-off cell against its approved plan; `status` shows the hand-off phase and checks the tenant's own wrap. `sb factory detach` builds the package from the running cell, adds the runbooks, verifies it, then detaches and tombstones the registry entry. Rehearsed in `tests/cell`: `TestAuditAgainstARunningCell` and `TestHandoffAndDetachRehearsal` (wrong region refused, execute before approvals refused, the planner can't approve, execute reaches TenantOwned, only retained paths are left, the package verifies, no factory path survives detach). The rehearsal found BUG-0024. The GCP hand-off target and an encrypted export file wait for SB-908.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
