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
| Status | Open |

## Why

An on-prem cell often reads a file share, and a demo needs documents without a vendor account.

## Acceptance criteria

- [ ] `docs: folder` reads a mounted directory: text, Markdown and PDF text; who may read each file comes from an ACL file beside it, and a file with no ACL is Unknown, never allowed
- [ ] Changes come from modification times with a cursor; a deleted file leaves the index at the next sync
- [ ] It passes the connector contract kit, and the self-hosted demo ships a small example share

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Connector
- [ ] ACL files
- [ ] Contract kit and demo share
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
