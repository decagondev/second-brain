---
title: "SB-3802: Ask about a file"
summary: "People can't ask about a file that isn't in the share."
---

# SB-3802: Ask about a file

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.1 |
| Stories | US-38.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-09, LAW-14 |
| Depends on | SB-2003 |
| Status | Done |

## Why

People can't ask about a file that isn't in the share.

## Acceptance criteria

- [x] The ask takes one file of up to 2 MB of a type the folder connector reads; its text is used for that ask only
- [x] The answer cites it as your file; the trace keeps only that there was a file, and the index none of it
- [x] Instruction-like text is flagged as in ingest; a larger file gives EINVAL

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API and agent
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `Request.File` (`ask.FileUpload`), `Agent.checkFile` and `Agent.fileChunks` (ingest's sanitiser, normaliser, chunker, secret and PII checks; at most 3 passages by the embedder, cited `file:NAME`, "Your file: NAME"), `Agent.FileText` wired to `folder.Text`, `Trace.Files`, `api.MaxAskBody`. The web UI's **Attach a file** sends one file with the next ask only. TestFileNameAndCheck, TestFileChunksFlagged, the running-cell test (cited, gone on Show passage, not indexed, 3 MB and .exe refused), web reach.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
