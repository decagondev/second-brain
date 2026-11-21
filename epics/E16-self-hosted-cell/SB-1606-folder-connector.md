---
title: "SB-1606: A folder connector for file shares and demos"
summary: "An on-prem cell often reads a file share, and a demo needs documents without a vendor account."
---

# SB-1606: A folder connector for file shares and demos

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.3 |
| Stories | US-16.1 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-01, LAW-03, LAW-13 |
| Depends on | [SB-1602](SB-1602-compose-layout.md) |
| Status | Done |

## Why

An on-prem cell often reads a file share, and a demo needs documents without a vendor account.

## Acceptance criteria

- [x] `docs: folder` reads a mounted directory: text, Markdown and PDF text; who may read each file comes from an ACL file beside it, and a file with no ACL is Unknown, never allowed
- [x] Changes come from modification times with a cursor; a deleted file leaves the index at the next sync
- [x] It passes the connector contract kit, and the self-hosted demo ships a small example share

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Connector
- [x] ACL files
- [x] Contract kit and demo share
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S15.3): `internal/adapters/connectors/folder` reads a directory opened as an `os.Root`, so a link can't leave it: `.txt`, `.md` and `.pdf` files (PDF text from its text operators, uncompressed or Flate, in the standard library; a file over 8 MiB or with no readable text is indexed by name). Who may read a file is in `FILE.acl` or the folder's `.acl` (`read EMAIL`, `read domain:DOMAIN`, `owner`, `label`); no ACL file, an unparsable one, a gone file or a late answer is Unknown. Changes compare the listing with the one the cursor names, kept by content hash under `state/connectors/KIND/` (eight kept), so a restart from an old cursor reports the same changes, a deleted file is a delete, and an ACL-only change is permission-only; a cursor whose listing is gone is an error, never a fresh start. It passes the contract kit with a fault-injecting view of the directory, and the law tests `TestLAW03_FolderUnknownIsNeverAllow` and `TestLAW13_FolderIndexIsDerived`. `examples/share` is the demo's share (operations for everyone at acme.example, a confidential forecast, a people folder ann can't read, a PDF, and a README with no ACL); the self-hosted demo mounts it and its smoke checks see answers cite it and never cite the people folder or the README.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
