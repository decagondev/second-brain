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
| Status | Done |

## Why

When a company stops using a box, nothing proves its data is gone.

## Acceptance criteria

- [x] `install.sh --retire CONFIG` stops the cell, removes its volumes, and overwrites then deletes the keys under `build/selfhost`
- [x] It writes a statement, signed with a key made for it and printed, naming what was destroyed, when, and the last audit anchor's hash
- [x] Sealed off-box copies stay openable only with the recovery kit; it refuses without `--yes-destroy`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Statement in Go with tests
- [x] Script
- [x] Runbook
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S18.3): `install.sh --retire [CONFIG] --yes-destroy` (`tools/selfhost/retire.sh`) runs the anchor job, takes the cell's audit export and checks it under the cell's own anchor key before anything is destroyed (`sb selfhost retire-statement --check`), and puts a copy off the box sealed to the recovery key. Then it removes the systemd units, the containers and every volume (by compose label and by name), overwrites then deletes every file under `build/selfhost` with `shred -u`, and checks that nothing is left. The statement (`internal/core/box/retire.go`, `sb-retire/v1`) names what was destroyed, what was kept, when, and the cell's own signed anchor on its last audit row, and is signed with an Ed25519 key made for it whose private half is never written; the public key is printed. `sb selfhost verify-retirement` checks it. Without `--yes-destroy` it lists what would go and stops. `TestLAW18_ARetiredBoxLeavesASignedStatement` with self-tests. Run for real on the demo box 2026-11-23: no volume, container or file under `build/selfhost` left; the statement verified and a changed field was refused; three kit shares rebuilt the recovery key, which opened the sealed audit export, whose chain verified under the cell's key (9 rows, 2 anchors).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
