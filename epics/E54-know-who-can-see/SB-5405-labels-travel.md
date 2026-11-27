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
| Status | Done |

## Why

Labels such as Confidential are read nowhere and shown nowhere, so labelled content goes wherever an answer goes.

## Acceptance criteria

- [x] Ingest reads Google Drive labels and front matter `sensitivity:` into `Chunk.Label` (public, internal, confidential, restricted)
- [x] Every answer, draft, share, download, print view, email and notification shows the highest label among its cites, and a note drafted from a labelled source carries it
- [x] Confidential content or above goes only to a person, never to a chat channel; a channel mention gets a DM and says so (LAW-32: a label never drops)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `Chunk.Label` and the Postgres column
- [x] Highest label on every way out
- [x] Channel guard in the chat apps
- [x] LAW-32 law test
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built, with LAW-32. Ingest reads front matter `sensitivity:` (`ingest.Sensitivity`) and only ever raises `Chunk.Label` with it; Drive labels were already read, and the Postgres `label` column already held it. `ask.CitedLabel` puts the highest cited label on every answer and draft (`label`), a published note carries its cites' label into its own front matter (`Stewards.citesLabel`, a missing cite counting as restricted), and the web shows it above the answer and in its download and print. The chat apps name it and refuse **Share in channel** for confidential or above (`ask.MayPostInChannel`, `MsgShareLabel`); `POST /v1/spaces/NAME/messages` takes the draft's cites and refuses the same (`Stewards.ChannelLabelOK`). LAW-32 in `laws/law32.go` with `TestSelfLabelGateIsCaught`; tested in TestLabelsReachAndTheSensitiveMapThroughARunningCell and TestConfidentialAnswerStaysWithThePerson.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
