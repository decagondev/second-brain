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
| Status | Done |

## Why

A recorded all-hands or call is invisible unless someone writes it up.

## Acceptance criteria

- [x] Audio and video files are transcribed by sb-cell(5) `model.transcribe`: the optional `transcribe` compose profile (whisper.cpp, weights pinned) or a gated provider in region
- [x] The result is type `transcript` with speakers, times and the AI mark, open to whoever can open the file
- [x] `transcribe.max_minutes` (default 120) caps a file; over it, or without `model.transcribe`, a recording is skipped and counted

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `transcribe` profile and gateway tier
- [x] Transcript chunks with times
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `ports.Document.Media` (audio and video files up to 200 MB) are transcribed by sb-cell(5) `model.transcribe` (`internal/adapters/model/transcribe`: whisper.cpp's `/inference` or OpenAI-compatible `/audio/transcriptions`, verbose JSON), only while `state/evals/transcribe-gated` names the release, which the evals job writes only when every word of each operator sample in `state/evals/transcribe/` is heard. `max_minutes` (default 120) is checked before (WAV header, else size) and after; spend counts at `per_minute_usd`; a month over budget transcribes nothing. Recordings are type `transcript` with `[HH:MM:SS]` lines and `read_by: transcribe`. `make selfhost-transcribe` (tools/selfhost/transcribe.sh) pins whisper.cpp, hears the sample and adds the `transcribe` compose profile. Speakers are named only where the provider gives them (whisper doesn't). Tested in TestTranscribe and TestRecordingsThroughARunningCell; a real whisper.cpp run stays a person's.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
