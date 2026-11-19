---
title: "SB-301: Tenant-scoped hybrid retrieval with ACL prefilter"
summary: "Retrieval narrows candidates; it never grants access."
---

# SB-301: Tenant-scoped hybrid retrieval with ACL prefilter

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.1 Retrieval |
| Stories | US-04.1 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-206](../E03-connectors-ingestion/SB-206-ingest-pipeline-chunk-embed-tag-provenance.md), [SB-108](../E01-tenancy-identity/SB-108-tenant-scoped-request-context.md) |
| Status | Done |

## Why

Retrieval narrows candidates; it never grants access.

## Acceptance criteria

- [x] Vector plus keyword retrieval over the tenant's own index only
- [x] Prefilter applied inside the query, not after
- [x] LAW-02 green on real indexes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retriever port and pgvector adapter
- [x] Keyword index
- [x] Fusion ranking
- [x] LAW-02 adversarial suite on real data
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.1, with SB-302 and SB-303):
- `ports.Retriever`, and `internal/adapters/postgres/index`, the pgvector index. It has a schema per tenant (`idx_TENANT`), is bound to one tenant and refuses any other scope or chunk. Search uses cosine distance and `tsvector` keyword rank, each with `acl && tokens` in the WHERE clause, and is tested against real Postgres 17 with pgvector.
- The in-memory index implements the same port for development and tests.
- `retrieve.Tokens` builds the asker's prefilter tokens, and `retrieve.Fuse` does reciprocal-rank fusion (k = 60).
- `TestLAW02_NoTenantCrossing`, red since W0, is green on the real pipeline. A self-test mutation (a cell that indexes another tenant's documents with a wrong prefilter) is caught. CI's `postgres` job now uses the `pgvector/pgvector:pg17` image.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
