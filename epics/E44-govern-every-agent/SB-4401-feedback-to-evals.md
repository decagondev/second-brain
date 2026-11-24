---
title: "SB-4401: Feedback becomes evals"
summary: "An agent's bad answers are rated and reported, but never become tests, so the same mistake ships again."
---

# SB-4401: Feedback becomes evals

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.1 |
| Stories | US-44.1 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-eval(5)`](../../manual/sb-eval.5.md) |
| Laws | LAW-01, LAW-06, LAW-07 |
| Depends on | SB-2904 |
| Status | Open |

## Why

An agent's bad answers are rated and reported, but never become tests, so the same mistake ships again.

## Acceptance criteria

- [ ] An unhelpful or reported answer on a named agent becomes a candidate case in its owner's inbox, with the asker's name removed
- [ ] The owner accepts it with the expected answer or cites, or dismisses it; accepted cases run in the next gate
- [ ] A candidate on documents the owner can't open can't be accepted

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Records
- [ ] API
- [ ] UI
- [ ] Gate
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
