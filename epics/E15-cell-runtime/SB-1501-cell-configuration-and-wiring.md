---
title: "SB-1501: Cell configuration and one composition root"
summary: "Every part of a cell is built and tested, but nothing builds the whole cell from one configuration."
---

# SB-1501: Cell configuration and one composition root

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.1 |
| Stories | US-15.1 |
| Manual | `sb-cell(5)` (new), [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-02, LAW-03, LAW-16 |
| Depends on | — |
| Status | Open |

## Why

Every part of a cell is built and tested, but nothing builds the whole cell from one configuration.

## Acceptance criteria

- [ ] A cell's configuration (`sb-cell(5)`: tenant, region, database, vault, adapters, listen addresses) is parsed strictly; unknown fields are errors
- [ ] `internal/cell/wire` builds every port and service of a cell from it in one place, local adapters or Postgres, with nothing built twice
- [ ] A missing or unreadable dependency stops the cell from starting; it never runs with a part switched off

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Configuration format and parser
- [ ] Composition root
- [ ] Start-up checks
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
