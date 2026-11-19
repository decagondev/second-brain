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
| Status | Open |

## Why

Retrieval narrows candidates; it never grants access.

## Acceptance criteria

- [ ] Vector plus keyword retrieval over the tenant's own index only
- [ ] Prefilter applied inside the query, not after
- [ ] LAW-02 green on real indexes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Retriever port and pgvector adapter
- [ ] Keyword index
- [ ] Fusion ranking
- [ ] LAW-02 adversarial suite on real data
- [ ] Write or extend the law tests: LAW-02
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
