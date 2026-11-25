---
title: "SB-5001: AI command centre"
summary: "Spend is split across budgets, caps and value pages, and nobody sees AI use by feature and tier in one place."
---

# SB-5001: AI command centre

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.1 |
| Stories | US-50.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4701, SB-4703 |
| Status | Open |

## Why

Spend is split across budgets, caps and value pages, and nobody sees AI use by feature and tier in one place.

## Acceptance criteria

- [ ] A costs page section and `GET /v1/admin/ai/usage` show 12 weeks of use and spend by space, feature, agent and tier, with budgets, caps and limits beside it
- [ ] Totals match the traces, and a CSV download matches the page
- [ ] People appear only as counts; any row under 5 shows as "fewer than 5"

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Usage query
- [ ] Costs page section
- [ ] CSV
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
