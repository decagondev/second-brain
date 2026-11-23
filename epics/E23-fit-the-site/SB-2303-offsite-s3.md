---
title: "SB-2303: Off-box copies to S3"
summary: "Off-box copies need a mounted NAS, not the bucket many sites already have."
---

# SB-2303: Off-box copies to S3

| Field | Value |
|---|---|
| Epic | [E23 Fit the Site](README.md) |
| Wave / slice | W22 / S22.1 |
| Stories | US-23.2 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Done |

## Why

Off-box copies need a mounted NAS, not the bucket many sites already have.

## Acceptance criteria

- [x] `SB_OFFSITE=s3://bucket/prefix` with a key pair in the vault sends the sealed dumps and backups to the bucket through the existing S3 adapter
- [x] `--doctor` checks the bucket and restore reads from it
- [x] Nothing unsealed leaves the box; the restore drill passes against a second S3 endpoint

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Scripts
- [x] Doctor check
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: done in S22.1. `SB_OFFSITE=s3://BUCKET/PREFIX` with `SB_OFFSITE_KEY_FILE`: the key pair goes to `vault://tenants/TENANT/offsite`, the backup job's sealed copy to the bucket (`backups[].s3`, sealed always, `s3.Bucket` with `Prefix` and `Plain`), and the scripts' dumps, kit and retirement copies through `sb backup push|pull|list|check` (`tools/selfhost/offsite.sh`). Push refuses anything not sealed or a small JSON record. `--doctor` checks the bucket; `restore.sh --dump s3:NAME` reads from it; the copy restores from a second store (`TestBackupToAnS3Bucket`). `TestLAW07_NothingUnsealedReachesTheBucket` with a self-test. Manuals: sb-selfhost(7), sb-cell(5), sb-tenant(5), sb(1).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
