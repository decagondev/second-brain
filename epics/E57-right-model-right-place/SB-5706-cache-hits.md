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
| Status | Done |

## Why

Nobody can tell whether the prompt cache is working, or which prefix keeps changing.

## Acceptance criteria

- [x] The share of input read from the prompt cache per space and agent this month, from traces, at `GET /v1/admin/costs/cache` and in the admin's week
- [x] A space or agent under a tenth with repeated asks is named, with the part of its prefix that changed between asks
- [x] No question, passage or memory appears in the report

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Cache shares from traces
- [x] Changed-prefix finder
- [x] Admin's week line
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. Each trace keeps `InputTokens`, `CachedTokens` (from `budget.Tally.AddTokens`, fed by the gateway) and `Prefix`, an 8-character hash of each part of the stable prefix (prompt, language, length, skill). `ask.BuildCacheHits` counts the month by space and agent at `GET /v1/admin/costs/cache`, marks a space or agent under a tenth over at least 2 asks as `low` with the parts that changed between asks, and the admin's week has a line for each. Tested in TestCacheHitsFromTraces (a prompt with a timestamp is named, no text in the report) and TestPrefixIsHashed.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
