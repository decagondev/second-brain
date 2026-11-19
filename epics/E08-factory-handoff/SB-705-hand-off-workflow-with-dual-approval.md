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
| Status | Open |

## Why

One workflow, matching `Handoff.tla`, with a plan hash both sides approve.

## Acceptance criteria

- [ ] States and guards identical to `Handoff.tla`; TLC run clean in CI with mutation checks
- [ ] Abort before cutover leaves the factory cell untouched
- [ ] Every step writes a `custody` audit row

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Run TLC on `Handoff.tla` in CI
- [ ] Controller state machine
- [ ] Plan hashing and approvals
- [ ] Snapshot, restore and index rebuild in region
- [ ] Cutover and DNS
- [ ] Write or extend the law tests: LAW-17, LAW-18
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
