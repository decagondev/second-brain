---
title: "sb-selfhost(7)"
man: "7"
summary: A whole cell on one machine the company owns, in containers that mirror how GCP runs a cell (ADR-0022; SB-1601, SB-1602).
---

# sb-selfhost(7)

## NAME

sb-selfhost — a self-hosted cell on one machine, in containers

## SYNOPSIS

```text
make images                 # build one image per service
make selfhost               # an example tenant's cell on this machine's Docker
make selfhost-stop          # stop it; its data stays in Docker volumes
make selfhost-clean         # stop it and delete its volumes
```

## DESCRIPTION

A self-hosted cell is the whole cell on one machine: a VM or an on-prem server that holds a company's data on its own hardware, or a laptop for demos and testing. It runs the same images and the same sb-cell(5) configuration as a cell on GCP, laid out the way GCP lays one out (ADR-0022): one container per service, each with only what it needs.

**Images** (SB-1601). `make images` builds `sb-api`, `sb-ingest`, `sb-jobs` and `sb-updater` as static Go binaries (`CGO_ENABLED=0`, `-trimpath`, no build id) and puts each in its own image on the distroless static base, pinned by digest, running as the non-root user, with no shell. The `api` image also carries the built web UI (`/sb/web`). Each image's id is written to `build/images/digests.txt`, and `tools/release/bundle --image NAME=sha256:…` records it in the release bundle as `image/NAME`, which is what the updater deploys by (sb-release(7)). Each binary has its own SBOM, `build/images/NAME.spdx.json`. `DOCKER` names the Docker command (on Windows with Docker Desktop, its `docker.exe`).

**The layout** (SB-1602), `deploy/selfhost/compose.yaml`:

| Container | Like on GCP | Networks | Holds |
|---|---|---|---|
| `proxy` (Caddy) | the HTTPS load balancer | `edge` | the only published port; TLS, here with Caddy's own local CA |
| `api` | the `api` Cloud Run service | `edge`, `data`, `egress` | its own vault: sign-in, SCIM, the control plane, the model, connector credentials (for live checks), the anchor and attestation keys |
| `ingest` | the `ingest` Cloud Run service | `data`, `egress` | its own vault: connector credentials only |
| `jobs` | the `jobs` Cloud Run job | `data`, `egress` | its own vault: connector credentials and the anchor key |
| `postgres` (pgvector) | Cloud SQL, private IP | `data` only, which has no route out | the database password; every record, the index and the audit log |
| `idp` (demo only) | the company's IdP | `edge` | the development IdP, for a demo with no IdP of its own |

`data` is an internal network: Postgres can't reach the internet and nothing outside reaches Postgres. Each service's secrets are a vault of its own in a volume of its own, owned by the service's user and readable by nobody else (the vault refuses a secret others can read), so `ingest` never holds the sign-in secret, as on GCP each service account reaches only its own secrets. The shared `state` volume holds what sb-cell(5) keeps in files (job finishes, cursors, the usage ledger, kill switches) and the applied manifest. The database's DSN is in the vault too (`database: "vault://tenants/TENANT/database"` in sb-cell(5)), so the configuration file holds no secret.

**What comes later.** The `updater` and its write-once store for anchors are S15.2 (SB-1603, SB-1604); until then the self-hosted cell has no WORM store, so its `anchor` job fails loudly, and it is upgraded by `make images` and `make selfhost`. The install for a real VM, with the company's own IdP and real TLS, is SB-1605.

**`make selfhost`** builds the images, writes a configuration, vaults and an example manifest under `build/selfhost/` (secrets made fresh, never committed), copies each vault into its volume with the right owner and mode, brings the containers up, waits for `/readyz` through the proxy, and provisions a demo user over SCIM. The cell is then at `https://cell.localhost:8443` and signs in through the demo IdP at `http://idp.localhost:8081`; both names resolve to this machine in a browser. The browser warns about Caddy's local certificate until it is trusted (`caddy trust` in the proxy container). `make cell` (sb-runtime(7)) stays the flat, no-container way to run the same binaries for development.

**The demo IdP** is `tools/devidp`. It has no passwords, so it refuses any address but loopback unless started with `--demo`, which the compose layout does, publishing it on this machine's loopback only. It never fronts a real cell.

## LAWS

[LAW-02](sb-laws.7.md#law-02): one machine, one tenant's cell. [LAW-07](sb-laws.7.md#law-07): each service reads only its own secrets, at call time, from files nobody else can read. [LAW-16](sb-laws.7.md#law-16): the cell runs its applied manifest, kept in its own state.

## EXAMPLES

```text
$ make selfhost
selfhost: acme-logistics is running at https://cell.localhost:8443 (demo IdP http://idp.localhost:8081)
selfhost: sign in as ann; make selfhost-stop stops it
```

## SEE ALSO

[sb-cell(5)](sb-cell.5.md), [sb-runtime(7)](sb-runtime.7.md), [sb-release(7)](sb-release.7.md), [sb-identity(7)](sb-identity.7.md)
