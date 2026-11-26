---
title: "SB-5405: Labels travel"
summary: "Labels such as Confidential are read nowhere and shown nowhere, so labelled content goes wherever an answer goes."
---

# SB-5405: Labels travel

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.2 |
| Stories | US-54.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-20, LAW-26, LAW-32 |
| Depends on | SB-5205, SB-3801 |
| Status | Open |

## Why

Labels such as Confidential are read nowhere and shown nowhere, so labelled content goes wherever an answer goes.

## Acceptance criteria

- [ ] Ingest reads Google Drive labels and front matter `sensitivity:` into `Chunk.Label` (public, internal, confidential, restricted)
- [ ] Every answer, draft, share, download, print view, email and notification shows the highest label among its cites, and a note drafted from a labelled source carries it
- [ ] Confidential content or above goes only to a person, never to a chat channel; a channel mention gets a DM and says so (LAW-32: a label never drops)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `Chunk.Label` and the Postgres column
- [ ] Highest label on every way out
- [ ] Channel guard in the chat apps
- [ ] LAW-32 law test
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
