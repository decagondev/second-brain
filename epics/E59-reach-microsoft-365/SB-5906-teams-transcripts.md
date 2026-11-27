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
| Status | Done |

## Why

What was said in Teams meetings never reaches the brain.

## Acceptance criteria

- [x] Teams transcripts (VTT) ingested through `folder.TranscriptText` as type `transcript`, with speakers and times
- [x] Open only to people who could open the meeting's recording
- [x] A meeting without a transcript is skipped

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Transcripts from Graph
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. The sharepoint connector lists each organizer's transcripts (`getAllTranscripts(...)/delta`) and fetches each as `SUBJECT (DATE) - Transcript` (type `transcript`) through `folder.TranscriptText`, open to the meeting's organizer and attendees, the live check reading the participants again; a meeting without a transcript is never listed. sb-cell(5) `microsoft.organizers`. Tested in TestTeamsTranscripts.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
