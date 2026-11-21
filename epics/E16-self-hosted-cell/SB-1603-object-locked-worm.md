---
title: "SB-1603: Write-once audit storage on the box"
summary: "LAW-11 needs anchors and the off-site backup copy that nobody can overwrite or delete for seven years; on one machine that is an object store with object lock."
---

# SB-1603: Write-once audit storage on the box

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.2 |
| Stories | US-16.1 |
| Manual | [`sb-audit(7)`](../../manual/sb-audit.7.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-11 |
| Depends on | [SB-1602](SB-1602-compose-layout.md) |
| Status | Done |

## Why

LAW-11 needs anchors and the off-site backup copy that nobody can overwrite or delete for seven years; on one machine that is an object store with object lock.

## Acceptance criteria

- [x] An S3-API write-once target (compliance-mode object lock, seven years) for anchors and the off-site backup, in the standard library (SigV4), no new module
- [x] The cell refuses a bucket without compliance-mode lock or with a shorter retention, as the GCP adapter refuses an unlocked bucket
- [x] Tested against a recorded fake; the compose layout uses a local object store, and the runbook says how to point it at one off the box

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] S3 write-once target
- [x] Lock checks
- [x] Fake and tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S15.2): `internal/adapters/backup/s3` is a `backup.Target` over the S3 API, signed with Signature Version 4 in the standard library (checked against AWS's own example), no new module. Before each write it reads the bucket's lock configuration and refuses one without object lock, in governance mode or under seven years (`ErrNotLocked`); every write is `If-None-Match: *` with a compliance-mode retain-until date, so a second write is `ErrExists`. `Enforced` writes a probe and tries to delete it, version and all, and refuses a store that lets the delete through. sb-cell(5) gained `worm` (and `{"worm": true}` in `backups`), with its key pair at `vault://tenants/TENANT/worm`; the `anchor` and `backup` jobs write through it. The compose layout runs versitygw (MinIO's images can no longer be pulled) on the `data` network, and the one-shot `wormsetup` makes, locks and proves the bucket at install. `internal/adapters/backup/s3/s3test` is the recorded fake, shared by the unit tests and the new law test `TestLAW11_AnchorsAreWriteOnceOnTheBox` (overwrites and deletes fail, weak or unenforced locks are refused). On Docker Desktop, `make selfhost-smoke` saw an anchor land in the store and the store refuse to delete a locked object. sb-selfhost(7) says how to point `worm` at a store off the box.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
