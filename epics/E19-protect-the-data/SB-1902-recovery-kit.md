---
title: "SB-1902: Recovery kit in shares"
summary: "The keys that make a restore possible live only on the box; the runbook says to back them up 'with the machine's own encrypted backup', which many sites don't have."
---

# SB-1902: Recovery kit in shares

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.1 |
| Stories | US-19.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | LAW-18 |
| Depends on | SB-1901 |
| Status | Done |

## Why

The keys that make a restore possible live only on the box; the runbook says to back them up "with the machine's own encrypted backup", which many sites don't have.

## Acceptance criteria

- [x] `install.sh --recovery-kit` splits the recovery private key and an archive of the box's keys into 5 shares, any 3 of which rebuild it (Shamir over GF(256))
- [x] Each share is a text page with its number and a checksum; a mistyped share fails its checksum, and any 2 shares reveal nothing
- [x] `restore.sh --from-kit` takes 3 shares; who holds a share is recorded, never the share

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Shamir in Go with tests
- [x] `sb recovery split` / `join`
- [x] Scripts and runbook
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S18.1): `install.sh --recovery-kit` (`tools/selfhost/kit.sh`) archives the box's keys (the vaults, the database and write-once secrets, the recovery key pair, the demo's release key) and `sb recovery split` seals it with a fresh 32-byte kit key (`offsite.SealWithKey`) and splits that key into five shares, any three of which rebuild it (`internal/core/shamir`, GF(256) with the AES polynomial). Splitting the kit key rather than the archive itself keeps each share to one typable page: its number, tenant, kit, 52 base32 characters in groups of five, and a checksum; the sealed archive goes to `build/selfhost/kit` and `SB_OFFSITE/kit/`. The record (`kit_ID.json`) names the custodians (`--custodian N=NAME`) and holds a 4-byte check of the key, never a share. The private recovery key is then shredded from the box; the doctor warns while it or any page is still there. `restore.sh --from-kit KIT SHARE SHARE SHARE` refuses a machine with keys, checks each page and the rebuilt key, writes the keys into `build/selfhost` and records which shares were used (`kit-used.jsonl`). `--new-key` replaces a lost kit. Run here end to end: a kit made on the demo box, the box wiped (`make selfhost-clean`), the keys rebuilt from shares 5, 1 and 3, the install run, the sealed off-box dump restored, and `sb audit verify` passed with 9 rows and 2 anchors; two shares were refused. Tests: `TestAnyThreeOfFive`, `TestTwoSharesRevealNothing` (every value of a byte stays possible), `TestSharePages` (a typo fails its checksum), `TestRecoverySplitJoin`, `TestLAW18_AnyThreeSharesAndNoTwo` with a self-test that catches a 2-of-5 kit.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
