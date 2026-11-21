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
| Manual | `sb-selfhost(7)` (new), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-02, LAW-07, LAW-16 |
| Depends on | [SB-1601](SB-1601-container-images.md) |
| Status | Open |

## Why

A company that wants its data on its own machine should get the same shape GCP runs: one container per service, each with only what it needs.

## Acceptance criteria

- [ ] `deploy/selfhost/compose.yaml` runs `api`, `ingest`, `jobs`, Postgres with pgvector, a write-once object store and a TLS proxy, each on its own network with health checks and restart policies; `updater` runs on a timer
- [ ] Each service gets only its own secrets as files (the Cloud Run service-account split, by mounts); only the proxy publishes a port
- [ ] `make selfhost` brings up an example tenant on Docker in one command; `make cell` stays the flat, no-container developer loop on the same configuration

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Compose file and networks
- [ ] Per-service secrets
- [ ] make selfhost
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
