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
| Status | Open |

## Why

LAW-11 needs anchors and the off-site backup copy that nobody can overwrite or delete for seven years; on one machine that is an object store with object lock.

## Acceptance criteria

- [ ] An S3-API write-once target (compliance-mode object lock, seven years) for anchors and the off-site backup, in the standard library (SigV4), no new module
- [ ] The cell refuses a bucket without compliance-mode lock or with a shorter retention, as the GCP adapter refuses an unlocked bucket
- [ ] Tested against a recorded fake; the compose layout uses a local object store, and the runbook says how to point it at one off the box

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] S3 write-once target
- [ ] Lock checks
- [ ] Fake and tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
