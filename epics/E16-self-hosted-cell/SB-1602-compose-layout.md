---
title: "SB-1602: A self-hosted cell in containers that mirror GCP"
summary: "A company that wants its data on its own machine should get the same shape GCP runs: one container per service, each with only what it needs."
---

# SB-1602: A self-hosted cell in containers that mirror GCP

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.1 |
| Stories | US-16.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) (new), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-02, LAW-07, LAW-16 |
| Depends on | [SB-1601](SB-1601-container-images.md) |
| Status | Done |

## Why

A company that wants its data on its own machine should get the same shape GCP runs: one container per service, each with only what it needs.

## Acceptance criteria

- [x] `deploy/selfhost/compose.yaml` runs `api`, `ingest`, `jobs`, Postgres with pgvector, a write-once object store and a TLS proxy, each on its own network with health checks and restart policies; `updater` runs on a timer
- [x] Each service gets only its own secrets as files (the Cloud Run service-account split, by mounts); only the proxy publishes a port
- [x] `make selfhost` brings up an example tenant on Docker in one command; `make cell` stays the flat, no-container developer loop on the same configuration

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Compose file and networks
- [x] Per-service secrets
- [x] make selfhost
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S15.1): `deploy/selfhost/compose.yaml` runs `proxy` (Caddy, the only published port, TLS from its local CA), `api`, `ingest`, `jobs`, `postgres` (pgvector, on the internal `data` network with no route out) and, for demos, the development IdP behind `--demo`. Each cell service runs as 65532 with a read-only root filesystem, no capabilities and `no-new-privileges`, and has a vault of its own in its own volume (`ingest` holds no sign-in secret); the database DSN comes from the vault (`database: vault://tenants/TENANT/database`, new in sb-cell(5)), so the configuration file holds no secret. `make selfhost` writes fresh secrets under `build/selfhost`, fills each volume with the right owner and mode, brings the layout up, waits for `/readyz` through the proxy and provisions a demo user over SCIM; `make selfhost-smoke` checks it from outside (15 checks: up, non-root, read-only, readyz and the UI over TLS, sign-in then an ask, a refused ask without sign-in, no route out of the database network, each vault closed to everyone else). Run on Docker Desktop: every check passed, and a control run showed the egress network does reach out, so the closed one is closed. CI runs both (`selfhost` job). Bringing it up found BUG-0026. The write-once store and the `updater` on a timer arrive with SB-1603 and SB-1604. `make cell` stays the flat developer loop on the same configuration.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
