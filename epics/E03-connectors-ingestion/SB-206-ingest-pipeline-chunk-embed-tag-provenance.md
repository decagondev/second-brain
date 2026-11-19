---
title: "SB-206: Ingest pipeline: chunk, embed, tag provenance"
summary: "Every chunk must know where it came from and which version it is."
---

# SB-206: Ingest pipeline: chunk, embed, tag provenance

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.3 Pipeline |
| Stories | US-03.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md) |
| Status | Done |

## Why

Every chunk must know where it came from and which version it is.

## Acceptance criteria

- [x] Each chunk has tenant, connector, source id, version, ACL prefilter and provenance hash
- [x] Pipeline is idempotent per (source id, version)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Queue consumer (`Pipeline.Sync` consumes the change feed; the durable per-connector cursor is SB-208)
- [x] Chunker with stable boundaries
- [x] Embedding adapter behind port
- [x] Provenance record
- [x] Write or extend the law tests: LAW-13
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: done.
- `internal/core/ingest`: `Normalise` and `Chunks` (paragraphs packed to 1,200 characters, long paragraphs cut at a space; boundaries depend only on the text); `Cite` and a derived `ChunkID`; `Pipeline.Ingest`, which skips an indexed version that is the same or newer and replaces all of a document's chunks in one write; `Sync`, where the last change per document wins, deletes are removed and permission changes are forced; and `Rebuild`, which drops the index and replays from the start of the change feed.
- New ports `Embedder` and `Index`, with a deterministic local hashing embedder and an in-memory index.
- Provenance on every chunk: tenant, cite, connector kind, source id, version, span, content SHA-256, ACL prefilter, release. Flags are empty until the sanitiser (SB-207).

LAW-13 has its first law test, red first, then green: ingest, re-ingest (no change), then drop and rebuild (same `(cite, chunk, ACL)` set).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
