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
| Status | Done |

## Why

The nightly database dump goes to the off-box share in plain text: whoever can read the NAS can read the whole brain.

## Acceptance criteria

- [x] The install makes a recovery key pair; only its public half stays on the box
- [x] Every file the box copies to `SB_OFFSITE` is sealed first: a fresh AES-256-GCM key per file, wrapped to the recovery key with X25519 and HKDF; the copy on the box is unchanged
- [x] `sb backup open` with the recovery key restores a sealed copy, and a changed byte fails to open

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `internal/core/seal` file format and tests
- [x] `sb backup seal` / `open`
- [x] dbdump.sh and the backup job's off-box copy
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S18.1): `internal/core/offsite` seals with the standard library only: a fresh AES-256-GCM key per file, wrapped to an X25519 recovery key with an ephemeral key and HKDF-SHA256 (`crypto/hkdf`, Go 1.24), the content in 64 KiB authenticated chunks with the last one marked and the header bound into the first, so a changed, dropped, reordered or truncated chunk, or another key, fails to open. The install makes the pair once (`sb backup keygen`; `recovery.pub` stays, the private half waits for the kit); `dbdump.sh` seals the off-box dump (`sb backup seal`, `NAME.dump.sealed`) and removes plain off-box dumps from before; `cell.json`'s off-box copy carries `seal`, and the backup job wraps that target in `offsite.Target`, so every object off the box is `NAME.sealed` and the copies on the box and in the write-once store are unchanged. `sb backup open` writes nothing unless the whole file opens. `SB_SEAL=off` turns it off, and the doctor warns. Copies the job wrote before sealing stay as they were (write-once). Run on the demo box: sealed dumps opened with the recovered key on a new machine. BUG-0029 was found and fixed on the way. Tests: `TestSealOpen` (sizes around the chunk boundary, a changed byte anywhere, truncation, another key), `TestBackupSealOpen`, `TestBackupSealsTheOffBoxCopy`, `TestLAW11_OffBoxCopiesAreSealed` with a self-test that catches an unauthenticated cipher.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
