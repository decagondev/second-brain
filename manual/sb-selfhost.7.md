---
title: "sb-selfhost(7)"
man: "7"
summary: A whole cell on one machine the company owns, in containers that mirror how GCP runs a cell, with write-once audit storage and signed upgrades (ADR-0022; SB-1601 to SB-1604).
---

# sb-selfhost(7)

## NAME

sb-selfhost — a self-hosted cell on one machine, in containers

## SYNOPSIS

```text
make images                 # build one image per service
make selfhost               # an example tenant's cell on this machine's Docker
make selfhost-update        # one upgrade: admit the channel's release, start it by digest
make selfhost-smoke         # check the running cell from outside
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
| `jobs` | the `jobs` Cloud Run job | `data`, `egress` | its own vault: connector credentials, the anchor key and the write-once store's key pair |
| `updater` (run on a timer) | the `updater` Cloud Run job | `data` | its own vault (empty unless the release store needs a pull token) and the release store, read-only |
| `worm` (versitygw) | the WORM Cloud Storage bucket | `data` only | every anchor and the backup's write-once copy, under compliance-mode object lock for seven years |
| `postgres` (pgvector) | Cloud SQL, private IP | `data` only, which has no route out | the database password; every record, the index and the audit log |
| `idp` (demo only) | the company's IdP | `edge` | the development IdP, for a demo with no IdP of its own |

`data` is an internal network: Postgres can't reach the internet and nothing outside reaches Postgres. Each service's secrets are a vault of its own in a volume of its own, owned by the service's user and readable by nobody else (the vault refuses a secret others can read), so `ingest` never holds the sign-in secret, as on GCP each service account reaches only its own secrets. The shared `state` volume holds what sb-cell(5) keeps in files (job finishes, cursors, the usage ledger, kill switches) and the applied manifest. The database's DSN is in the vault too (`database: "vault://tenants/TENANT/database"` in sb-cell(5)), so the configuration file holds no secret.

**Write-once storage** (SB-1603). LAW-11 needs anchors and a backup copy that nobody, the cell included, can overwrite or delete for seven years. On GCP that is a bucket with a locked retention policy; on the box it is an S3 store with **compliance-mode object lock**, here versitygw on its own volume, reachable only on the `data` network. The cell talks to it with the standard S3 API, signed with Signature Version 4 in the standard library (`internal/adapters/backup/s3`), with the key pair at `vault://tenants/TENANT/worm` (sb-cell(5), `worm`). Before every write it reads the bucket's lock configuration and refuses a bucket without object lock, in governance mode (which an administrator can bypass) or with less than seven years. Every write carries `If-None-Match: *`, so a second write to a name is refused, and its own retain-until date in compliance mode. At install the one-shot `wormsetup` container makes the bucket, locks it, and proves the store enforces the lock: it writes a probe object and tries to delete it, version and all; a store that lets the delete through is refused, whatever its configuration claims. The probe stays, locked, as the proof. To keep a copy **off the box**, point `worm.endpoint` at an S3 store on another machine or site with compliance-mode lock, set `worm.offsite: true`, and list `{"worm": true}` in `backups`, so the backup's write-once copy is also its off-site one (sb-audit(7), Backups).

**Upgrades** (SB-1604). The updater has to deploy by digest, but a container that holds the Docker socket holds the whole machine, so no container does. The work is split. The `updater` container, run once per check, pulls the channel's release, admits it with the four checks of sb-release(7), attests each image digest, and records what each service should run under `state/deploy/services/`; that is all it can do. Then the host step, `tools/selfhost/update.sh`, run by `make selfhost-update` or by the `sb-selfhost-update.timer` systemd unit hourly (`deploy/selfhost/systemd/`), reads the record and refuses, before changing anything, a reference that isn't pinned by digest, an image the cell didn't attest, or an image this machine doesn't hold. It writes the digests, and nothing else, to `build/selfhost/images.env`, brings the services up on them, checks that each container runs exactly its digest, and records that under `state/deploy/running/`. The verifier's attestation reads `running/`, so it reports what the containers really run (sb-release(7), Attestation). A release the updater doesn't admit, unsigned, ungated or signed by another key, stops the host step at the first step: nothing changes, the unit fails, and the updater raises a Sev-1 `release_unsigned` alert. `make selfhost` publishes a **demo release** of this machine's own images, signed with a key made for this machine (`build/selfhost/release-key`, by `tools/selfhost/devrelease`) that only the demo configuration trusts; a real install trusts the factory's release identity instead (SB-1605).

**Running it on a box.** `deploy/selfhost/systemd/sb-selfhost.service` brings the layout up at boot on the admitted digests, and `sb-selfhost-update.timer` runs the upgrade hourly. Both expect the repository (or a release's `deploy/` and `tools/selfhost/`) at `/opt/second-brain`. The install for a real VM, with the company's own IdP and real TLS, is SB-1605.

**`make selfhost`** builds the images, writes a configuration, vaults and an example manifest under `build/selfhost/` (secrets made fresh, never committed), publishes the demo release, copies each vault into its volume with the right owner and mode, brings the containers up, makes and proves the write-once bucket, waits for `/readyz` through the proxy, and provisions a demo user over SCIM. Run `make selfhost-update` once to move the services onto the admitted digests. **`make selfhost-smoke`** checks the result from outside: the services' users and read-only roots, sign-in and an ask through TLS, the database network's missing route out, each vault's owner and mode, an anchor in the write-once store, a refused delete, no Docker socket in any container, each service on its admitted digest, and a release signed with another key changing nothing. The cell is then at `https://cell.localhost:8443`; `/login` signs in as the demo user through the demo IdP at `http://idp.localhost:8081`; both names resolve to this machine in a browser. The browser warns about Caddy's local certificate until it is trusted (`caddy trust` in the proxy container). `make cell` (sb-runtime(7)) stays the flat, no-container way to run the same binaries for development.

**The demo IdP** is `tools/devidp`. It has no passwords, so it refuses any address but loopback unless started with `--demo`, which the compose layout does, publishing it on this machine's loopback only. It never fronts a real cell.

## LAWS

[LAW-02](sb-laws.7.md#law-02): one machine, one tenant's cell. [LAW-06](sb-laws.7.md#law-06): the cell runs only a release it admitted, by digest, and an unadmitted one changes nothing. [LAW-11](sb-laws.7.md#law-11): anchors and the backup's write-once copy can't be overwritten or deleted for seven years. [LAW-07](sb-laws.7.md#law-07): each service reads only its own secrets, at call time, from files nobody else can read. [LAW-16](sb-laws.7.md#law-16): the cell runs its applied manifest, kept in its own state.

## EXAMPLES

```text
$ make selfhost
selfhost: acme-logistics is running at https://cell.localhost:8443 (demo IdP http://idp.localhost:8081)
selfhost: sign in as ann at https://cell.localhost:8443/login; make selfhost-stop stops it
$ make selfhost-update
selfhost-update: running rel_2026-11-21_96f9cb on images pinned by digest
```

## SEE ALSO

[sb-cell(5)](sb-cell.5.md), [sb-audit(7)](sb-audit.7.md), [sb-runtime(7)](sb-runtime.7.md), [sb-release(7)](sb-release.7.md), [sb-identity(7)](sb-identity.7.md)
