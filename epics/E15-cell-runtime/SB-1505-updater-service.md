---
title: "SB-1505: The updater job: releases pulled and admitted"
summary: "The updater's logic exists against the GCP fake; it needs a binary and a local deployer."
---

# SB-1505: The `updater` job: releases pulled and admitted

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.3 |
| Stories | US-15.1 |
| Manual | [`sb-runtime(7)`](../../manual/sb-runtime.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-06, LAW-08 |
| Depends on | [SB-1502](SB-1502-api-service.md) |
| Status | Done |

## Why

The updater's logic exists against the GCP fake; it needs a binary and a local deployer.

## Acceptance criteria

- [x] `cmd/sb-updater` reads the channel, admits the bundle, attests and deploys each service by digest, local or GCP
- [x] An unsigned or ungated bundle changes nothing and raises a Sev-1

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Binary
- [x] Local deployer
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.3): `cmd/sb-updater --config FILE` builds `runtime.Updater` from the configuration alone (`wire.NewUpdater`): the store (`file://` or the factory's `https://` store with the pull token), the channel pointer (`channels/CHANNEL`, read by both stores), the one trusted release identity (Ed25519 or cosign) and the deployer. `internal/adapters/deploy/localfs` records what each service runs and refuses an image the cell hasn't attested, as Binary Authorization does; `gcp.CloudRun` deploys on GCP with the updater's own token from the metadata server (`gcp.MetadataToken`, tested against a fake). A release that isn't admitted changes nothing and sends a Sev-1 `release_unsigned` (`tests/cell`, `TestUpdaterPullsAdmitsAndDeploys`: every service deployed by digest; a second run changes nothing; an unsigned and an ungated release are both refused with nothing changed and one Sev-1 each). The verifier's attestation is now wired to the deployer and the cell's attestation key, and the control plane's identity (`vault://tenants/TENANT/control-plane`) guards the verifier. The GCP deploy path is tested against fakes only; a real project is SB-903 and SB-908.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
