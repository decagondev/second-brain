---
title: "SB-5706: Cache hits you can see"
summary: "Nobody can tell whether the prompt cache is working, or which prefix keeps changing."
---

# SB-5706: Cache hits you can see

| Field | Value |
|---|---|
| Epic | [E57 Right Model, Right Place](README.md) |
| Wave / slice | W56 / S56.2 |
| Stories | US-57.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-5605, SB-5606, SB-5006 |
| Status | Open |

## Why

Nobody can tell whether the prompt cache is working, or which prefix keeps changing.

## Acceptance criteria

- [ ] The share of input read from the prompt cache per space and agent this month, from traces, at `GET /v1/admin/costs/cache` and in the admin's week
- [ ] A space or agent under a tenth with repeated asks is named, with the part of its prefix that changed between asks
- [ ] No question, passage or memory appears in the report

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Cache shares from traces
- [ ] Changed-prefix finder
- [ ] Admin's week line
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
