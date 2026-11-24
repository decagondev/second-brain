---
title: "SB-3701: Meeting transcripts"
summary: "Meetings decide things, but their transcripts can't be found or cited."
---

# SB-3701: Meeting transcripts

| Field | Value |
|---|---|
| Epic | [E37 Capture Conversations](README.md) |
| Wave / slice | W36 / S36.1 |
| Stories | US-37.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-01, LAW-09 |
| Depends on | SB-1606 |
| Status | Done |

## Why

Meetings decide things, but their transcripts can't be found or cited.

## Acceptance criteria

- [x] The folder connector reads .vtt and .srt: each cue's speaker and text become a line, and the cite carries its start time
- [x] Masking applies as to any document; the brain records no meeting
- [x] A malformed cue is skipped and counted

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Folder connector
- [x] Cites
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

The folder connector reads .vtt and .srt cue by cue, each start time a heading, broken cues skipped and counted

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
