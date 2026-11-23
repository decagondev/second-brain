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
| Status | Open |

## Why

Off-box copies need a mounted NAS, not the bucket many sites already have.

## Acceptance criteria

- [ ] `SB_OFFSITE=s3://bucket/prefix` with a key pair in the vault sends the sealed dumps and backups to the bucket through the existing S3 adapter
- [ ] `--doctor` checks the bucket and restore reads from it
- [ ] Nothing unsealed leaves the box; the restore drill passes against a second S3 endpoint

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Scripts
- [ ] Doctor check
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
