---
title: "SB-1702: Who to ask: owners on citations"
summary: "Chunks already carry their document's owner for barriers, but an answer never says who to ask."
---

# SB-1702: Who to ask: owners on citations

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.1 |
| Stories | US-17.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-01, LAW-12, LAW-22 |
| Depends on | — |
| Status | Proposed |

## Why

Chunks already carry their document's owner for barriers, but an answer never says who to ask.

## Acceptance criteria

- [ ] Each allowed cite gains `owner`: the owner's email and `title` from the directory, only for an active user; omitted otherwise
- [ ] The answer gains `people`: the owners of its allowed cites, most cited first, at most three; the UI shows "Ask …, owner of …"
- [ ] An owner is never shown for a refused cite, and a deprovisioned owner is never suggested
- [ ] No activity or expertise profile is built; ownership at the source is the only signal

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Owner lookup on allowed cites
- [ ] UI and Slack reply
- [ ] Tests, including a deprovisioned owner
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Proposed 2026-11-22 with W16 ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
