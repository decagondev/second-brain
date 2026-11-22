---
title: "SB-1901: Off-box copies sealed"
summary: "The nightly database dump goes to the off-box share in plain text: whoever can read the NAS can read the whole brain."
---

# SB-1901: Off-box copies sealed

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.1 |
| Stories | US-19.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | — |
| Status | Open |

## Why

The nightly database dump goes to the off-box share in plain text: whoever can read the NAS can read the whole brain.

## Acceptance criteria

- [ ] The install makes a recovery key pair; only its public half stays on the box
- [ ] Every file the box copies to `SB_OFFSITE` is sealed first: a fresh AES-256-GCM key per file, wrapped to the recovery key with X25519 and HKDF; the copy on the box is unchanged
- [ ] `sb backup open` with the recovery key restores a sealed copy, and a changed byte fails to open

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `internal/core/seal` file format and tests
- [ ] `sb backup seal` / `open`
- [ ] dbdump.sh and the backup job's off-box copy
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
