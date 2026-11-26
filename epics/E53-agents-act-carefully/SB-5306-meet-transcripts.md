---
title: "SB-5306: Meeting transcripts from Drive"
summary: "Meeting transcripts sit in Drive as plain documents, without speakers or action items."
---

# SB-5306: Meeting transcripts from Drive

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.2 |
| Stories | US-53.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-02 |
| Depends on | SB-3701, SB-4203 |
| Status | Done |

## Why

Meeting transcripts sit in Drive as plain documents, without speakers or action items.

## Acceptance criteria

- [x] The Drive connector reads Google Meet transcript documents as transcripts, with speakers, times and the meeting
- [x] Action items are offered to SB-4203 and count only once their person confirms them
- [x] The document's access stays the ceiling; a non-attendee without access never sees it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Transcript detection on the recorded fake
- [x] Speakers and action items
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. The Drive connector knows a Google Meet transcript by the name Meet gives it (`MEETING (DATE) - Transcript`, `gdrive.IsMeetTranscript`) and keeps its `Speaker: words` lines, dropping Meet's timestamp lines (`gdrive.MeetText`); `ask.TypeOf` makes such a Doc a `transcript`, so search's `type=transcript`, the decisions draft (SB-3702) and its action items (SB-4203) take it as they take a `.vtt` file. The Doc's permissions stay its access. Tested in TestMeetTranscripts and TestMeetTranscriptIsATranscript.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
