---
title: "SB-5606: What it saved"
summary: "Nobody can see what reuse, batching, caching and the economy tier are worth."
---

# SB-5606: What it saved

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.2 |
| Stories | US-56.2 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-11 |
| Depends on | SB-3505, SB-5006, SB-5603, SB-5604, SB-5605 |
| Status | Open |

## Why

Nobody can see what reuse, batching, caching and the economy tier are worth.

## Acceptance criteria

- [ ] The cost statement and the admin's week gain a "saved" line: reuse, batching, caching and the economy tier, each in dollars this month
- [ ] `GET /v1/admin/costs/saved` gives the same as JSON, worked out from traces against the full price
- [ ] A month with no savings says so

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Savings from traces
- [ ] Statement and admin's week lines
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
