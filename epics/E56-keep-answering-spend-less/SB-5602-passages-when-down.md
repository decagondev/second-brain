---
title: "SB-5602: Passages when the model is down"
summary: "When no model is available the ask box fails outright, though search still works."
---

# SB-5602: Passages when the model is down

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.1 |
| Stories | US-56.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | SB-5601, SB-3207 |
| Status | Done |

## Why

When no model is available the ask box fails outright, though search still works.

## Acceptance criteria

- [x] With no gated model available, an ask returns the best passages the person can open, cited, under "No answer written: the model is unavailable"
- [x] Briefs and routines wait and run when a model comes back
- [x] The health report and the notice banner say why; nothing is labelled as an answer

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Passages-only response
- [x] Deferred scheduled runs
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `Agent.composeOrPassages` turns a model that can't answer into `notice: model_unavailable`, no claims and up to 5 `passages` the asker can open (masked search results); `TestModelDownIsUnavailable` now asserts that. While the fallback state is down the notify job leaves its drafting parts for a later tick (`NotifyJob`), and `GET /v1/notice` shows the banner with `model_down`. Web: the passages and the line above them.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
