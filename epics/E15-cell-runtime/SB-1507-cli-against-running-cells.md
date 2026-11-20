---
title: "SB-1507: The CLI commands that need a running cell"
summary: "`sb audit tail`, `sb audit export`, `sb eval diff`, `sb factory handoff` and `sb factory detach` answer `ENOTIMPL` today."
---

# SB-1507: The CLI commands that need a running cell

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.4 |
| Stories | US-15.1 |
| Manual | [`sb(1)`](../../manual/sb.1.md), [`sb-factory(1)`](../../manual/sb-factory.1.md), [`sb-audit(1)`](../../manual/sb-audit.1.md) |
| Laws | LAW-11, LAW-17, LAW-18 |
| Depends on | [SB-1506](SB-1506-apply-to-a-running-cell.md) |
| Status | Open |

## Why

`sb audit tail`, `sb audit export`, `sb eval diff`, `sb factory handoff` and `sb factory detach` answer `ENOTIMPL` today.

## Acceptance criteria

- [ ] Each command works against a local cell and leaves `planned`
- [ ] Hand-off and detach run their game-day rehearsals against a running cell

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Commands
- [ ] Rehearsals
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
