---
title: "SB-6102: Recordings become transcripts"
summary: "A recorded all-hands or call is invisible unless someone writes it up."
---

# SB-6102: Recordings become transcripts

| Field | Value |
|---|---|
| Epic | [E61 Every Format, Every Language](README.md) |
| Wave / slice | W60 / S60.1 |
| Stories | US-61.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-06, LAW-28, LAW-33 |
| Depends on | SB-3701, SB-1710, SB-6101 |
| Status | Open |

## Why

A recorded all-hands or call is invisible unless someone writes it up.

## Acceptance criteria

- [ ] Audio and video files are transcribed by sb-cell(5) `model.transcribe`: the optional `transcribe` compose profile (whisper.cpp, weights pinned) or a gated provider in region
- [ ] The result is type `transcript` with speakers, times and the AI mark, open to whoever can open the file
- [ ] `transcribe.max_minutes` (default 120) caps a file; over it, or without `model.transcribe`, a recording is skipped and counted

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `transcribe` profile and gateway tier
- [ ] Transcript chunks with times
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
