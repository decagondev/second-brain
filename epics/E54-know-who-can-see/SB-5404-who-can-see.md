---
title: "SB-5404: Who can see this"
summary: "Nobody can tell how far a cite, note or shared answer actually reaches."
---

# SB-5404: Who can see this

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.1 |
| Stories | US-54.1 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-02 |
| Depends on | SB-1703 |
| Status | Done |

## Why

Nobody can tell how far a cite, note or shared answer actually reaches.

## Acceptance criteria

- [x] `GET /v1/documents/{id}/audience` and **Who can see this** beside a cite, a note and a shared answer give the number of people who can open it now
- [x] Groups behind it are named only when they are the person's own or the space's; others are counted, never named
- [x] For a shared answer, the count is of people who can open every cite

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Audience from the live ACL and the directory
- [x] Web: **Who can see this**
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `Agent.Audience` at `GET /v1/documents/{id}/audience` and `GET /v1/audience?doc=…` (repeated, for a shared answer's cites) asks the source as each active person and counts those it allows for every document (`unknown` apart); groups are named only when they are the caller's or the space's, the rest counted. A document the caller can't open, or one held from them, is `EPERM` like one that doesn't exist. Web: **Who can see this** beside each cite (`Sharing.tsx`).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
