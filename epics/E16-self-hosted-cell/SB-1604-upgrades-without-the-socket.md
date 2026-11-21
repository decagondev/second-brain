---
title: "SB-1604: Upgrades without giving the cell the Docker socket"
summary: "The updater must deploy by digest, but a container holding the Docker socket holds the whole machine."
---

# SB-1604: Upgrades without giving the cell the Docker socket

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.2 |
| Stories | US-16.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), `sb-selfhost(7)` |
| Laws | LAW-06 |
| Depends on | [SB-1602](SB-1602-compose-layout.md) |
| Status | Open |

## Why

The updater must deploy by digest, but a container holding the Docker socket holds the whole machine.

## Acceptance criteria

- [ ] The updater admits the release and records the digests to run (the local deployer, SB-1505); a host step run by a systemd timer reads them and runs only those digests, refusing a tag or an unattested image
- [ ] No container mounts the Docker socket; an unsigned or ungated release changes nothing and raises a Sev-1
- [ ] The verifier's attestation reports what the containers really run

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Host apply step and timer
- [ ] Refusals
- [ ] Attestation check
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
