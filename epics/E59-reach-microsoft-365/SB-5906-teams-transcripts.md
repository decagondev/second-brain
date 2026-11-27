---
title: "SB-5906: Teams meeting transcripts"
summary: "What was said in Teams meetings never reaches the brain."
---

# SB-5906: Teams meeting transcripts

| Field | Value |
|---|---|
| Epic | [E59 Reach Microsoft 365](README.md) |
| Wave / slice | W58 / S58.2 |
| Stories | US-59.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-09 |
| Depends on | SB-3701, SB-5901 |
| Status | Open |

## Why

What was said in Teams meetings never reaches the brain.

## Acceptance criteria

- [ ] Teams transcripts (VTT) ingested through `folder.TranscriptText` as type `transcript`, with speakers and times
- [ ] Open only to people who could open the meeting's recording
- [ ] A meeting without a transcript is skipped

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Transcripts from Graph
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
