---
title: "SB-605: Friendly-tenant onboarding, unaided"
summary: "The self-serve goal is proven, not claimed."
---

# SB-605: Friendly-tenant onboarding, unaided

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.2 Onboarding |
| Stories | US-07.1 |
| Manual | [`sb-onboarding(7)`](../../manual/sb-onboarding.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-604](../E07-rollout-ops/SB-604-sb-onboarding-7-day-0-to-90-runbook.md) |
| Status | In review |

## Why

The self-serve goal is proven, not claimed.

## Acceptance criteria

- [ ] A friendly tenant completes day 0 to 14 with zero platform-team steps
- [ ] Every question they asked becomes a manual fix

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Run onboarding
- [ ] Observe silently
- [ ] Log gaps as tickets
- [ ] Write or extend the law tests: LAW-01, LAW-02, LAW-03
- [ ] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.2): this ticket is a human exercise: a friendly tenant runs day 0 to 14 unaided while we watch silently. It gets tooling, not code. The runbook is checked by `tools/check/runbook.py`, the gates call `sb tenant verify` (SB-601) and the pilot report (SB-311), and questions go into the retro template, where each becomes a ticket or a manual fix. The LAW-01 to LAW-03 checks on their real data are `sb tenant verify`.

**Left (people):** run it with a friendly tenant on a deployed cell (W7), observe, and log every gap.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
