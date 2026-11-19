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
| Status | Open |

## Why

Every chunk must know where it came from and which version it is.

## Acceptance criteria

- [ ] Each chunk has tenant, connector, source id, version, ACL prefilter and provenance hash
- [ ] Pipeline is idempotent per (source id, version)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Queue consumer
- [ ] Chunker with stable boundaries
- [ ] Embedding adapter behind port
- [ ] Provenance record
- [ ] Write or extend the law tests: LAW-13
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
