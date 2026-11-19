---
title: "SB-1106: Field-level allow-lists for restricted sources"
summary: "Structured systems (HRIS, accounting, CRM) hold fields that must never become chunks."
---

# SB-1106: Field-level allow-lists for restricted sources

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.3 People and Finance sources |
| Stories | US-11.3 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), `sb-space(5)` (new) |
| Laws | [LAW-07](../../manual/sb-laws.7.md#law-07), LAW-20 (new, W8) |
| Depends on | [SB-1005](../E10-department-spaces/SB-1005-ingest-routes-and-labels-every-document.md) |
| Status | Open |

## Why

Structured systems (HRIS, accounting, CRM) hold fields that must never become chunks.

## Acceptance criteria

- [ ] A connector declares fields; a space's allow-list names which may be ingested; anything else is dropped before chunking
- [ ] Dropping is logged by field name and count, never value
- [ ] A connector that can't declare its fields can't feed a restricted space

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Field declaration in the contract
- [ ] Allow-list in the manifest
- [ ] Ingest filter
- [ ] Tests
- [ ] Write or extend the law tests: LAW-07, LAW-20
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
