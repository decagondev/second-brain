---
title: "SB-1705: Answer once: replies become notes"
summary: "A steward's answer lives in a reply or a chat thread and is found once; the next person asks again."
---

# SB-1705: Answer once: replies become notes

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.2 |
| Stories | US-17.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-04, LAW-05, LAW-09, LAW-13, LAW-15 |
| Depends on | [SB-1704](SB-1704-ask-the-stewards.md) |
| Status | Done |

## Why

A steward's answer lives in a reply or a chat thread and is found once; the next person asks again.

## Acceptance criteria

- [x] A steward publishes a reply as a note: a Markdown file and its ACL file under `notes/` in the folder share, through the tool bus as `note-publish` on the folder connector's `act`
- [x] The note is `Ran` only with a receipt, its path and SHA-256 read back after the write; the publish binds the note's hash
- [x] Readers default to the space's members and can only be narrowed; before publishing, the steward sees how many cites are readable by fewer people than the note and confirms
- [x] `act` writes only `.md` and `.acl` under `notes/`, opened as a root it can't leave; the rest of the share stays read-only; no model writes a note
- [x] The next ingest indexes the note, and an ask that matches it gets a cited answer

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notes directory in the self-hosted layout
- [x] `act` on the folder connector, contract kit
- [x] Publish from the steward's reply
- [x] Tests and a smoke check
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.2): with the manifest's `notes: true` (which needs `docs: folder`), a steward publishes a reply as a note through the tool bus as `note-publish` (a `Call` now carries policy context; Cedar refuses `note-publish` to anyone who isn't a steward of the note's space). The folder connector's `act` opens `notes/` alone as a root, writes `notes/SPACE/qs_ID.md` and then its ACL file, never over another note, reads both back, and returns `PATH@sha256:HEX` as the receipt. Readers default to the space's active members by email and can only be narrowed; the preview counts cites some readers can't open, and publishing needs that count confirmed. The steward writes the note's title; the body is their reply. In the self-hosted layout `notes/` is a separate mount, writable only into `api` (a smoke check inspects the mounts). `make proofs` green. Tests: `TestLAW05_NotesRanMeansTheFileOnTheShare` (real bus and connector, an outsider refused), `TestNotes` (paths, overwrite, payloads, the rest of the share untouched), the core note test, and the running cell: the next ask is answered from the note with its cite. The self-hosted smoke run waits for S16.5's run on Docker Desktop.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
