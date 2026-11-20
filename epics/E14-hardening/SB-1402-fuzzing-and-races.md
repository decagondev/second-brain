---
title: "SB-1402: Fuzz the parsers and run every test under the race detector"
summary: "Parsers that take outside input are where a crash or a wrong accept hides; concurrent code is where a race hides."
---

# SB-1402: Fuzz the parsers and run every test under the race detector

| Field | Value |
|---|---|
| Epic | [E14 Hardening](README.md) |
| Wave / slice | W13 / S13.2 |
| Stories | US-14.1 |
| Manual | as the change needs (manual first, P2) |
| Laws | LAW-03, LAW-07 |
| Depends on | [SB-1401](SB-1401-fresh-eyes-review.md) |
| Status | Done |

## Why

Parsers that take outside input are where a crash or a wrong accept hides; concurrent code is where a race hides.

## Acceptance criteria

- [x] Go fuzz tests for every parser of outside input: the tenant manifest, SCIM bodies, action payloads and their hash, the reorg snapshot, and connector responses
- [x] `make fuzz` runs each fuzz target for a bounded time, and its corpus is checked in
- [x] `go test -race` passes over every package in `make check`
- [x] Every crash or wrong accept found is fixed with a regression test

## Subtasks

- [x] Update the man pages the change touches first (manual first, P2)
- [x] Fuzz targets and corpus
- [x] Race detector in `make check`
- [x] Fix what they find
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S13.2): six fuzz targets (`FuzzCheck`, `FuzzCanonical`, `FuzzParse`, `FuzzChunksAndSecrets`, `FuzzMatch`, `FuzzUsers`) and `make fuzz`; `make test` runs under `-race`, and now includes `tests/spaces` and `tests/groupsync`, which `make check` hadn't run before. Each target ran for 45 seconds to 2 minutes; `FuzzCheck` found two ways an accepted manifest read back differently (BUG-0021), both fixed, with their inputs kept as regression seeds. The race detector found nothing. Connector responses are decoded by `encoding/json` into fixed shapes and covered by the contract kit, so they have no target of their own (docs/04-testing-and-assurance.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
