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
| Status | Open |

## Why

Parsers that take outside input are where a crash or a wrong accept hides; concurrent code is where a race hides.

## Acceptance criteria

- [ ] Go fuzz tests for every parser of outside input: the tenant manifest, SCIM bodies, action payloads and their hash, the reorg snapshot, and connector responses
- [ ] `make fuzz` runs each fuzz target for a bounded time, and its corpus is checked in
- [ ] `go test -race` passes over every package in `make check`
- [ ] Every crash or wrong accept found is fixed with a regression test

## Subtasks

- [ ] Update the man pages the change touches first (manual first, P2)
- [ ] Fuzz targets and corpus
- [ ] Race detector in `make check`
- [ ] Fix what they find
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
