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
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-04, LAW-11, LAW-12, LAW-17, LAW-18 |
| Depends on | [SB-1603](SB-1603-object-locked-worm.md), [SB-1604](SB-1604-upgrades-without-the-socket.md) |
| Status | In review |

## Why

The self-hosted cell is for a company that keeps its data on its own hardware, and for demos and testing; it needs an install, an IdP and a runbook.

## Acceptance criteria

- [x] An install script for one Linux VM: Docker, the compose layout, TLS, the cell's own root key on the box (never leaving it), backups to three copies with one off the box
- [x] Sign-in through the company's OIDC IdP, or a bundled IdP for demos and air-gapped sites; the development IdP is refused outside loopback
- [x] A runbook covers install, upgrade, backup and restore, and what LAW-17 means on one machine (a signed access statement instead of an IAM check); `sb tenant verify --cell` is green on a fresh install

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Install script
- [x] Identity options
- [x] Runbook and verify
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S15.3): `tools/selfhost/selfhost.sh` is now the one setup path, the demo with no configuration and an install with `SB_CONFIG` (`deploy/selfhost/install.env.example`): TLS by ACME, the company's files or Caddy's CA; sign-in by the company's OIDC IdP or the bundled one, the development IdP refused off loopback; a folder share; the backup's off-box directory; the release store and identity. `tools/selfhost/install.sh` sets up a Linux VM or server from a checkout or an offline install bundle (`make selfhost-bundle`), and installs the systemd units for the layout, hourly upgrades and a nightly database dump (`tools/selfhost/dbdump.sh`), here and off the box. Every key is made on the box and stays there. The bundled IdP, sb-idp(1) (`cmd/sb-idp`, `internal/adapters/oidc/bundled`), signs in with PBKDF2-hashed passwords, refuses another client or redirect before showing its form, limits failures per address and per client behind the proxy, and provisions users in the cell over SCIM; it is tested against the cell's own OIDC client, and the demo now uses it instead of signing in with no password. LAW-17 on one machine: the cell is in tenant custody from install, its factory paths are listed in `state/iam`, and the monthly `access` job writes a statement whose hash goes into the audit log and so into the next anchor (`TestAccessStatement`). The runbook is `docs/runbooks/selfhost.md`. On Docker Desktop the demo passes every smoke check (34, twice in a row), `sb tenant verify --cell` is green against it, and the install path ran from a configuration file. **In review:** a run of `install.sh` on a real Linux VM with the company's IdP and certificate, which needs a machine.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
