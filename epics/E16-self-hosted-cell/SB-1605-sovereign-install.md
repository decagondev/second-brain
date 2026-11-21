---
title: "SB-1605: A sovereign install on a VM or an on-prem server"
summary: "The self-hosted cell is for a company that keeps its data on its own hardware, and for demos and testing; it needs an install, an IdP and a runbook."
---

# SB-1605: A sovereign install on a VM or an on-prem server

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.3 |
| Stories | US-16.1 |
| Manual | `sb-selfhost(7)`, [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-04, LAW-11, LAW-12, LAW-17, LAW-18 |
| Depends on | [SB-1603](SB-1603-object-locked-worm.md), [SB-1604](SB-1604-upgrades-without-the-socket.md) |
| Status | Open |

## Why

The self-hosted cell is for a company that keeps its data on its own hardware, and for demos and testing; it needs an install, an IdP and a runbook.

## Acceptance criteria

- [ ] An install script for one Linux VM: Docker, the compose layout, TLS, the cell's own root key on the box (never leaving it), backups to three copies with one off the box
- [ ] Sign-in through the company's OIDC IdP, or a bundled IdP for demos and air-gapped sites; the development IdP is refused outside loopback
- [ ] A runbook covers install, upgrade, backup and restore, and what LAW-17 means on one machine (a signed access statement instead of an IAM check); `sb tenant verify --cell` is green on a fresh install

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Install script
- [ ] Identity options
- [ ] Runbook and verify
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
