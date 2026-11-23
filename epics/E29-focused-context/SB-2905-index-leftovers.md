---
title: "SB-2905: Index leftovers"
summary: "The index can keep chunks the source no longer has until the next rebuild."
---

# SB-2905: Index leftovers

| Field | Value |
|---|---|
| Epic | [E29 A Focused Context](README.md) |
| Wave / slice | W28 / S28.2 |
| Stories | US-29.2 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md), [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb(1)`](../../manual/sb.1.md) |
| Laws | LAW-13, LAW-11 |
| Depends on | — |
| Status | Open |

## Why

The index can keep chunks the source no longer has until the next rebuild.

## Acceptance criteria

- [ ] The nightly job counts chunks of documents the source no longer lists, and chunks no space holds in a cell with spaces; health warns above zero
- [ ] `sb index prune --dry-run` lists them by cite and `sb index prune` removes them with an audit row
- [ ] A missed tombstone is counted and pruned; nothing a source still lists is touched

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Job and health
- [ ] CLI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
