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
| Status | Done |

## Why

Chunks already carry their document's owner for barriers, but an answer never says who to ask.

## Acceptance criteria

- [x] Each allowed cite gains `owner`: the owner's email and `title` from the directory, only for an active user; omitted otherwise
- [x] The answer gains `people`: the owners of its allowed cites, most cited first, at most three; the UI shows "Ask …, owner of …"
- [x] An owner is never shown for a refused cite, and a deprovisioned owner is never suggested
- [x] No activity or expertise profile is built; ownership at the source is the only signal

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Owner lookup on allowed cites
- [x] UI and Slack reply
- [x] Tests, including a deprovisioned owner
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.1): each cite gains `owner` (email and SCIM title) through `ask.ActiveOwner`, only when exactly one active directory user has the owner's email, looked up per answer and never cached; a failing directory shows no owner rather than failing the answer. The answer gains `people`, the owners of its cites, most cited first, at most three. Cites now carry the indexed title. The web UI shows "Ask …, owner of …" under an answer, and the Slack reply ends with the same line. Tests: `TestLAW12_OwnersAreActiveAndAllowed` (its self-test catches a lookup that ignores deprovisioning), `TestAskNamesOwnersOfAllowedCites` (a refused document's owner and a deprovisioned owner are never shown), the Slack render and UI tests.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
