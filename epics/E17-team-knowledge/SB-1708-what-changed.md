---
title: "SB-1708: What changed in my spaces"
summary: "Ingest knows every new version, and nobody is told what changed in the things they work on."
---

# SB-1708: What changed in my spaces

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.4 |
| Stories | US-17.4 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | — |
| Status | Proposed |

## Why

Ingest knows every new version, and nobody is told what changed in the things they work on.

## Acceptance criteria

- [ ] The index records when it first saw each version; versions written by a rebuild are marked as such
- [ ] `GET /v1/changes?since=7d[&space=…]` lists documents with a new version in the window, in the person's spaces, each checked live, newest first
- [ ] The UI shows What changed this week on the empty answer page; the weekly report counts changes per space
- [ ] A rebuild never shows as a week of changes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] First-seen column and rebuild mark
- [ ] Endpoint and UI
- [ ] Tests, including a rebuild
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Proposed 2026-11-22 with W16 ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
