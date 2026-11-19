---
title: "SB-705: Hand-off workflow with dual approval"
summary: "One workflow, matching `Handoff.tla`, with a plan hash both sides approve."
---

# SB-705: Hand-off workflow with dual approval

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.2 Hand-off |
| Stories | US-08.2 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-704](../E08-factory-handoff/SB-704-root-key-re-wrap-and-key-custody-transfer.md), [SB-412](../E05-actions-approvals-audit/SB-412-audit-and-approval-backups-3-2-1-and-restore.md) |
| Status | In review |

## Why

One workflow, matching `Handoff.tla`, with a plan hash both sides approve.

## Acceptance criteria

- [x] States and guards identical to `Handoff.tla`; TLC run clean in CI with mutation checks
- [x] Abort before cutover leaves the factory cell untouched
- [x] Every step writes a `custody` audit row

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Run TLC on `Handoff.tla` in CI
- [x] Controller state machine
- [x] Plan hashing and approvals
- [ ] Snapshot, restore and index rebuild in region (the `Target` port's `Move`; the game day's restore exists, SB-613; a cloud Target needs a running cell, W7)
- [ ] Cutover and DNS (the `Target` port's `Cutover`; the revocation and its LAW-17 check are built)
- [x] Write or extend the law tests: LAW-17, LAW-18
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.2b): `internal/core/handoff.Controller` is `Handoff.tla` in code. `TestStatesMatchTheModel` reads the model and compares its phases, its single-phase actions (guard and effect) and `InFlight` with the controller's, and TLC already checks the model in CI (`make proofs`). A plan's hash is `ph_` plus the SHA-256 of its RFC 8785 form. A new plan voids every approval, approvals bind a hash, and the planner can't approve for the factory. Every step writes a `kind: custody` audit row before it acts, and stores its phase only after it succeeds. Steps go Provision, then the verified re-wrap (SB-704), Move, Verify, then Cutover, which revokes every unretained path and re-checks that none remain (LAW-17). Last comes the factory wrap's destruction under the live guard, and the custody record changes to `tenant`. Abort runs from `Planned` to `Verified` and restores `Managed` exactly, and Detach revokes every path. The LAW-17 and LAW-18 hand-off law tests now run on the real controller and are green, and the ratchet is empty again. A self-test shows the random runs reach all nine phases.

**Left:** `sb factory handoff` in the CLI and a cloud `Target` (move, verify and cutover against a running cell, W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
