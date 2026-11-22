---
title: "SB-1907: Retire a box with proof"
summary: "When a company stops using a box, nothing proves its data is gone."
---

# SB-1907: Retire a box with proof

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.3 |
| Stories | US-19.3 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-11, LAW-18 |
| Depends on | SB-1901 |
| Status | Open |

## Why

When a company stops using a box, nothing proves its data is gone.

## Acceptance criteria

- [ ] `install.sh --retire CONFIG` stops the cell, removes its volumes, and overwrites then deletes the keys under `build/selfhost`
- [ ] It writes a statement, signed with a key made for it and printed, naming what was destroyed, when, and the last audit anchor's hash
- [ ] Sealed off-box copies stay openable only with the recovery kit; it refuses without `--yes-destroy`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Statement in Go with tests
- [ ] Script
- [ ] Runbook
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
