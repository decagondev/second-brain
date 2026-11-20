---
title: "SB-1101: Template catalogue as data and sb space init"
summary: "A company should pick departments, not design them."
---

# SB-1101: Template catalogue as data and `sb space init`

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.1 Catalogue |
| Stories | US-11.1 |
| Manual | `sb-departments(7)` (new), `sb-space(5)` (new), [`sb(1)`](../../manual/sb.1.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-1001](../E10-department-spaces/SB-1001-spaces-in-the-manifest-and-sb-tenant-check.md) |
| Status | Done |

## Why

A company should pick departments, not design them.

## Acceptance criteria

- [x] Thirteen templates shipped in the release bundle as data, never code
- [x] `sb space init --template people` writes a valid `spaces:` entry into the manifest
- [x] A company can rename, merge or drop any template, and `custom` covers the rest

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Catalogue files
- [x] CLI
- [x] Bundle pinning
- [x] Examples
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S11.1a): the new page sb-space(1); sb-departments(7) says the catalogue is data; sb-release(7) adds the bundle's `catalogue` hash; sb(1) lists `sb space`. The thirteen templates are `internal/core/manifest/catalogue.json` (name, label, default sources, default agents, retention, eval suite), embedded and parsed by `manifest.ParseCatalogue`, which refuses an unknown field, a bad name or label, a template without `ask`, a retention under 30 days and a catalogue without `custom`; the Go table is gone. A test holds the file and sb-departments(7)'s table to each other. The bundle tool pins `catalogue` (SHA-256), so a template change is a new release. `sb space init FILE --template T --members … [--name] [--stewards]` adds one space under the existing block `spaces:` key or a new one, keeps the rest of the file byte for byte, scopes it from the template's sources the manifest has connectors for, and writes only if the whole file then checks; `sb space templates` prints the catalogue. Renaming is `--name`, merging is one space with both scopes, and dropping is not listing it. The new law test `TestLAW16_TheCatalogueIsPinned` edits a random template field and requires a different release id; a bundle without the catalogue is caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
