---
title: "SB-701: Final `sb-factory(7)` and the hand-off runbook"
summary: "Manual first: the hand-off is specified before it's built."
---

# SB-701: Final `sb-factory(7)` and the hand-off runbook

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.1 Custody model |
| Stories | US-08.1 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-604](../E07-rollout-ops/SB-604-sb-onboarding-7-day-0-to-90-runbook.md) |
| Status | Open |

## Why

Manual first: the hand-off is specified before it's built.

## Acceptance criteria

- [ ] Every `sb factory` command has synopsis, options, exit codes and examples
- [ ] Runbook for tenant owners and factory operators, reviewed by security

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Finalise command grammar
- [ ] Write runbook
- [ ] Review with a friendly tenant's IT
- [ ] Write or extend the law tests: LAW-17, LAW-18
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
