---
title: "SB-2904: Eval cases that went stale"
summary: "An eval case keeps expecting a document version that has moved on."
---

# SB-2904: Eval cases that went stale

| Field | Value |
|---|---|
| Epic | [E29 A Focused Context](README.md) |
| Wave / slice | W28 / S28.2 |
| Stories | US-29.2 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | — |
| Depends on | — |
| Status | Open |

## Why

An eval case keeps expecting a document version that has moved on.

## Acceptance criteria

- [ ] A case from a wrong citation records the version it expects; a new version or a gone document marks it check this case on the steward screen
- [ ] The nightly run reports it apart from failures; nothing is deleted on its own
- [ ] A new version marks the case and it isn't a regression

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Record
- [ ] UI and job
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
