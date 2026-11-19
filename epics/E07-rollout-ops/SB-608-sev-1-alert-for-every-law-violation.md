---
title: "SB-608: Sev-1 alert for every law violation"
summary: "A law that can break silently is not a law."
---

# SB-608: Sev-1 alert for every law violation

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.3 Operate |
| Stories | US-07.4 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-07](../../manual/sb-laws.7.md#law-07), [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-607](../E07-rollout-ops/SB-607-per-tenant-observability.md) |
| Status | In review |

## Why

A law that can break silently is not a law.

## Acceptance criteria

- [x] Every law with a production check has an alert and a runbook
- [ ] Alert test fires each one in staging (in process today: `TestEachRuleFires`; staging with W7)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Alert rules
- [ ] Runbooks
- [ ] Fire drill
- [x] Write or extend the law tests: LAW-01, LAW-02, LAW-06, LAW-07, LAW-11, LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.3a): `internal/core/alerts` has one Sev-1 rule for each law with a production check (LAW-01, 02, 04, 06, 07, 08, 11, 12, 16, 18), each with a runbook in `docs/runbooks/`. A test reads sb-laws(7) and fails if a law with a **Watch** line has no rule or no runbook. Every rule fires on its own signal and not without it, alerts carry ids and the runbook path, and every rule's metric is catalogued.

**Left:** firing each alert in staging through the real webhook (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
