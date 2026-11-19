---
title: "SB-1005: Ingest routes and labels every document"
summary: "A document must know which spaces hold it and how sensitive it is before anyone can ask about it."
---

# SB-1005: Ingest routes and labels every document

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.2 Space policy |
| Stories | US-10.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-13](../../manual/sb-laws.7.md#law-13), LAW-20 (new, W8) |
| Depends on | [SB-1004](SB-1004-cedar-schema-v2-spaces-labels-narrowing.md), [SB-206](../E03-connectors-ingestion/SB-206-ingest-pipeline-chunk-embed-tag-provenance.md) |
| Status | Open |

## Why

A document must know which spaces hold it and how sensitive it is before anyone can ask about it.

## Acceptance criteria

- [ ] Each document version gets its spaces (from scopes) and a label: the source's label if it has one, else the space default, else `internal`
- [ ] A document matched by a restricted space and a non-restricted space takes the stricter label
- [ ] Changing a scope or label re-routes on the next sync; a rebuild gives the same result (LAW-13)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Router
- [ ] Label mapping from Drive labels and Slack channels
- [ ] Rebuild test
- [ ] Write or extend the law tests: LAW-13, LAW-20
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
