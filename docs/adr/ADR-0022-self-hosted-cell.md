---
title: "ADR-0022: A self-hosted cell runs in containers that mirror GCP, on one machine"
summary: "The self-hosted cell: one container per service with Compose, Postgres with pgvector, an object-locked store for WORM, keys on the box, upgrades without the Docker socket; the flat make cell stays for development."
---

# ADR-0022: A self-hosted cell runs in containers that mirror GCP, on one machine

| Status | Date | Deciders |
|---|---|---|
| Proposed (plan, W15) | 2026-11-21 | Platform lead (the user) |

## Context

The hosting review (2026-11-21) found that hosting sits behind ports (ADR-0018), and that a single machine is the cheapest host and the one a company can own outright. The user wants a self-hosted cell: a VM or an on-prem server that holds a company's cell as sovereign data storage, and the same thing for demos and testing. Two layouts were possible: containers that mirror how GCP runs a cell, or every service as a plain process on one machine. `make cell` (SB-1508) already runs the flat one for development.

## Decision

- **Containers, mirroring GCP, are the supported self-hosted cell.** Docker Compose runs one container per service (`api`, `ingest`, `jobs`; `updater` on a timer), as Cloud Run runs one service per account. Each gets only its own secrets, mounted as files (the service-account split); only a TLS proxy publishes a port; Postgres and the object store sit on an internal network.
- **The flat layout stays the developer loop.** `make cell` runs the same binaries on the same sb-cell(5) configuration as processes, for quick tests. Nothing differs but the supervisor, so a bug found in one is a bug in the other.
- **Data:** Postgres 17 with pgvector in a container, its volume on the box.
- **WORM (LAW-11):** an S3-compatible object store with compliance-mode object lock for seven years, on the box for demos and pointed off the box for production; the cell refuses a store without the lock.
- **Keys (LAW-18):** the cell's own root key on the box (the `local` Keys adapter), never leaving it; an HSM or TPM can replace it later behind the same port.
- **Upgrades (LAW-06):** no container holds the Docker socket. The updater admits a release and records digests; a host step on a systemd timer runs exactly those digests.
- **Access (LAW-17):** on one machine the factory's access paths are logins on the box, which no API lists; a hand-off to a self-hosted cell is a signed access statement and an audit, not an IAM check. The runbook says so.
- **Identity:** the company's own OIDC IdP, or a bundled one for demos and air-gapped sites. The development IdP stays loopback-only.

## Alternatives rejected

- **Flat processes as the product.** Simpler, but every service shares one user, one set of secrets and one network, which is nothing like the GCP cell the laws were checked on.
- **Kubernetes (k3s) on the box.** It would mirror GCP more closely but adds a cluster to run; ADR-0011 keeps Kubernetes out of v1.
- **Giving the updater the Docker socket.** It would make upgrades simpler and the updater root on the machine.

## Consequences

- One image per service (SB-1601) serves GCP and the box alike.
- A self-hosted cell holds every law the GCP cell holds except LAW-17, which it holds by statement; the manual says so plainly.
- Demos and tests run the real cell on a laptop with `make selfhost`.
