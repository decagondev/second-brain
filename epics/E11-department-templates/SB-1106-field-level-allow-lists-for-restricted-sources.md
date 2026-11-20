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
| Status | Done |

## Why

Structured systems (HRIS, accounting, CRM) hold fields that must never become chunks.

## Acceptance criteria

- [x] A connector declares fields; a space's allow-list names which may be ingested; anything else is dropped before chunking
- [x] Dropping is logged by field name and count, never value
- [x] A connector that can't declare its fields can't feed a restricted space

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Field declaration in the contract
- [x] Allow-list in the manifest
- [x] Ingest filter
- [x] Tests
- [x] Write or extend the law tests: LAW-07, LAW-20
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S11.3a): sb-connector(3) gained "Fields", sb-ingest(7) "Fields" and sb-space(5) the `fields` allow-list and rule 12. `ports.Document.Fields`, `ports.Field` and the optional `ports.FieldDeclarer` let a structured connector return named fields and declare which are sensitive. Ingest builds such a document's text itself from the allowed fields only: declared and not sensitive, or sensitive and named by a holding space's `fields` allow-list, which makes the document `restricted`; undeclared fields are never allowed. Dropped fields go to `FieldsDropped` by name only, and a source that can't declare its fields is never ingested (its old chunks are deleted), so it can't feed a restricted space. Spaces carry `fields` through the manifest, plan, format, schema and registry. The mock source can be structured. Law tests `TestLAW07_StructuredFieldsOnlyWhenAllowed` and `TestLAW20_AllowedSensitiveFieldsAreRestricted` run random records, declarations and allow-lists through the real pipeline; a source that hides which fields are sensitive is caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
