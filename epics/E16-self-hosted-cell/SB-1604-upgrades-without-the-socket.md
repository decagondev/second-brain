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
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-06 |
| Depends on | [SB-1602](SB-1602-compose-layout.md) |
| Status | Done |

## Why

The updater must deploy by digest, but a container holding the Docker socket holds the whole machine.

## Acceptance criteria

- [x] The updater admits the release and records the digests to run (the local deployer, SB-1505); a host step run by a systemd timer reads them and runs only those digests, refusing a tag or an unattested image
- [x] No container mounts the Docker socket; an unsigned or ungated release changes nothing and raises a Sev-1
- [x] The verifier's attestation reports what the containers really run

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Host apply step and timer
- [x] Refusals
- [x] Attestation check
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S15.2): no container holds the Docker socket. The `updater` runs as a one-shot container (compose profile `update`) that admits the channel's release and records each service's digest through the `local` deployer, which on a self-hosted cell covers `api`, `ingest`, `jobs` and `updater` (`runtime.SelfHosted`, no `web`). The host step `tools/selfhost/update.sh` (`make selfhost-update`, or `deploy/selfhost/systemd/sb-selfhost-update.timer` hourly) refuses an unpinned reference, an unattested image or one the machine doesn't hold before changing anything, starts exactly those digests through `build/selfhost/images.env`, checks each container's image, and records it under `state/deploy/running/`; the `local` deployer now reports `running/` first, so the attestation says what the containers run (`TestRunningIsWhatTheHostStarted`). An unadmitted release stops the step at once and raises the Sev-1 `release_unsigned` (`TestUpdaterPullsAdmitsAndDeploys` covers the alert with the local deployer). `make selfhost` publishes a demo release of the machine's own images signed with a key made for the machine (`tools/selfhost/devrelease`, tested to be admitted only with that key). On Docker Desktop the smoke checks saw no socket in any container, each service on its admitted digest, and a release signed with another key refused with nothing changed.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
