---
title: "sb-selfhost(7)"
man: "7"
summary: A whole cell on one machine the company owns, in containers that mirror how GCP runs a cell, with write-once audit storage, signed upgrades and an install for a VM or on-prem server (ADR-0022; SB-1601 to SB-1606).
---

# sb-selfhost(7)

## NAME

sb-selfhost — a self-hosted cell on one machine, in containers

## SYNOPSIS

```text
make images                 # build one image per service
make selfhost               # the demo: an example tenant's cell on this machine's Docker
sudo tools/selfhost/install.sh [--install-docker] [--check] CONFIG   # an install on a Linux VM or server
make selfhost-bundle        # an offline install bundle: every image and the install files
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
| `proxy` (Caddy) | the HTTPS load balancer | `edge`, `egress` | the only published ports (HTTPS, and HTTP for ACME's challenge); the install's TLS. The cell's own services reach the cell's public name through it |
| `api` | the `api` Cloud Run service | `edge`, `data`, `egress` | its own vault: sign-in, SCIM, the control plane, the model, connector credentials (for live checks), the anchor and attestation keys |
| `ingest` | the `ingest` Cloud Run service | `data`, `egress` | its own vault: connector credentials only |
| `jobs` | the `jobs` Cloud Run job | `data`, `egress` | its own vault: connector credentials, the anchor key and the write-once store's key pair |
| `updater` (run on a timer) | the `updater` Cloud Run job | `data`, `egress` | its own vault (empty unless the release store needs a pull token) and the release store, read-only |
| `worm` (versitygw) | the WORM Cloud Storage bucket | `data` only | every anchor and the backup's write-once copy, under compliance-mode object lock for seven years |
| `postgres` (pgvector) | Cloud SQL, private IP | `data` only, which has no route out | the database password; every record, the index and the audit log |
| `idp` (profile `bundled`) | the company's IdP | `edge` | sb-idp(1): its users' password hashes and its signing key, in its own volume; behind the proxy at `/idp` |
| `devidp` (profile `dev`) | none | `egress` | the development IdP, no passwords, on loopback only |

`data` is an internal network: Postgres can't reach the internet and nothing outside reaches Postgres. Each service's secrets are a vault of its own in a volume of its own, owned by the service's user and readable by nobody else (the vault refuses a secret others can read), so `ingest` never holds the sign-in secret, as on GCP each service account reaches only its own secrets. The shared `state` volume holds what sb-cell(5) keeps in files (job finishes, cursors, the usage ledger, kill switches) and the applied manifest. The database's DSN is in the vault too (`database: "vault://tenants/TENANT/database"` in sb-cell(5)), so the configuration file holds no secret.

**Write-once storage** (SB-1603). LAW-11 needs anchors and a backup copy that nobody, the cell included, can overwrite or delete for seven years. On GCP that is a bucket with a locked retention policy; on the box it is an S3 store with **compliance-mode object lock**, here versitygw on its own volume, reachable only on the `data` network. The cell talks to it with the standard S3 API, signed with Signature Version 4 in the standard library (`internal/adapters/backup/s3`), with the key pair at `vault://tenants/TENANT/worm` (sb-cell(5), `worm`). Before every write it reads the bucket's lock configuration and refuses a bucket without object lock, in governance mode (which an administrator can bypass) or with less than seven years. Every write carries `If-None-Match: *`, so a second write to a name is refused, and its own retain-until date in compliance mode. At install the one-shot `wormsetup` container makes the bucket, locks it, and proves the store enforces the lock: it writes a probe object and tries to delete it, version and all; a store that lets the delete through is refused, whatever its configuration claims. The probe stays, locked, as the proof. To keep a copy **off the box**, point `worm.endpoint` at an S3 store on another machine or site with compliance-mode lock, set `worm.offsite: true`, and list `{"worm": true}` in `backups`, so the backup's write-once copy is also its off-site one (sb-audit(7), Backups).

**Upgrades** (SB-1604). The updater has to deploy by digest, but a container that holds the Docker socket holds the whole machine, so no container does. The work is split. The `updater` container, run once per check, pulls the channel's release, admits it with the four checks of sb-release(7), attests each image digest, and records what each service should run under `state/deploy/services/`; that is all it can do. Then the host step, `tools/selfhost/update.sh`, run by `make selfhost-update` or by the `sb-selfhost-update.timer` systemd unit hourly (`deploy/selfhost/systemd/`), reads the record and refuses, before changing anything, a reference that isn't pinned by digest, an image the cell didn't attest, or an image this machine doesn't hold (a release whose images live in a registry is pulled by digest, which Docker checks against the content). It writes the digests, and nothing else, to `build/selfhost/compose.env` (`SB_*_IMAGE`), brings the services up on them, checks that each container runs exactly its digest, and records that under `state/deploy/running/`. The verifier's attestation reads `running/`, so it reports what the containers really run (sb-release(7), Attestation). A release the updater doesn't admit, unsigned, ungated or signed by another key, stops the host step at the first step: nothing changes, the unit fails, and the updater raises a Sev-1 `release_unsigned` alert. `make selfhost` publishes a **demo release** of this machine's own images, signed with a key made for this machine (`build/selfhost/release-key`, by `tools/selfhost/devrelease`) that only the demo configuration trusts; a real install trusts the factory's release identity instead (SB-1605).

**Install** (SB-1605). One script sets up every self-hosted cell, `tools/selfhost/selfhost.sh`; with no configuration it is the demo, and with `SB_CONFIG=FILE` it is an install, from a file of settings (`deploy/selfhost/install.env.example`):

| Setting | Meaning |
|---|---|
| `SB_TENANT`, `SB_MANIFEST`, `SB_POLICY` | the tenant, its manifest (sb-tenant(5)), and its own policy, optional |
| `SB_HOST`, `SB_PORT`, `SB_HTTP_PORT`, `SB_BIND` | where people reach the cell: on a VM its DNS name, 443, 80 and `0.0.0.0` |
| `SB_TLS` | `acme` (a public name; Caddy gets and renews the certificate), `files` (`SB_TLS_CERT`, `SB_TLS_KEY`, and `SB_TLS_CA` for a company CA) or `internal` (Caddy's own CA, for a lab) |
| `SB_IDP` | `company` (`SB_OIDC_ISSUER`, `SB_OIDC_CLIENT_ID`, `SB_OIDC_SECRET_FILE`), `bundled` (sb-idp(1)), or `dev`, which is refused unless `SB_BIND` is loopback |
| `SB_SHARE` | a directory for `docs: folder` (sb-connector(3)), mounted read-only into the cell; with the manifest's `notes: true`, its `notes/` folder alone is mounted writable into `api`, for stewards' notes (SB-1705) |
| `SB_OFFSITE` | the backup's off-box copy: a directory on another machine, mounted here |
| `SB_RELEASE_STORE`, `SB_RELEASE_TRUSTED`, `SB_RELEASE_PULL_TOKEN_FILE`, `SB_RELEASE_REGISTRY` | where releases come from (the factory's `https://` store, or a directory releases are copied into on an air-gapped site), the one identity that signs them, and where their images live |

`sudo tools/selfhost/install.sh CONFIG` does the rest on a Linux VM or server: it checks the machine (Docker Engine and the compose plugin, installed from the distribution with `--install-docker`; systemd; an off-box directory that isn't on the root disk), loads the images of an **install bundle** (`make selfhost-bundle`: every image the layout runs and the files the install reads, for a machine with no Go, no Node and no internet) or builds them where Go is present, runs `selfhost.sh` with the configuration, applies the first release, and installs and starts the systemd units: `sb-selfhost.service` (the layout at boot, on the admitted digests), `sb-selfhost-update.timer` (hourly upgrades) and `sb-selfhost-dbdump.timer` (the nightly database dump). `--check` checks the machine and the configuration and changes nothing.

**Keys stay on the box.** Every secret and key is made on the machine by the install, once, and never leaves it: the database password, the OIDC client secret (or the company's, copied in), the SCIM token, the control plane's token, the anchor and attestation key seeds, the write-once store's key pair, and the bundled IdP's signing key. They sit in the per-service vaults under `build/selfhost/vault` (mode 0600) and in the volumes the install copies them into. Back that directory up with the machine's own encrypted backup or not at all; the release store and the factory never see any of it.

**Sign-in.** With `SB_IDP=company` the cell signs in with the company's OIDC IdP, which must send the cell's tenant in a `tenant` claim and the user's directory id in `sub` (sb-identity(7)) and provision users over SCIM. With `SB_IDP=bundled` it uses sb-idp(1), behind the proxy at `https://HOST/idp`: passwords as PBKDF2 hashes, failed sign-ins limited, and users added with `sb-idp user add`, which provisions them in the cell over SCIM so the directory and the IdP agree. The development IdP is refused by the install outright, and by `selfhost.sh` anywhere but loopback. When the proxy's certificate comes from Caddy's own CA or a company CA, `selfhost.sh` gives the services that CA's certificate (only the certificate) at `/etc/sb/ca`, so the cell can reach its own sign-in.

**Backups: three copies, one off the box.** The nightly `backup` job (sb-audit(7), Backups) writes the audit log and approvals to `state/backups` on the box, to the write-once store, and to `SB_OFFSITE`, so there are three copies on two kinds of media with one off the machine; it refuses to run with fewer. The nightly database dump (`tools/selfhost/dbdump.sh`, `pg_dump`) keeps fourteen nights of every other record on the box (`build/selfhost/db-dumps`) and off it (`SB_OFFSITE/db`). The index can also be rebuilt from the sources (LAW-13).

**LAW-17 on one machine.** A self-hosted cell is in the company's custody from the start (`custody.tenant: true`). There is no IAM to enumerate, so the install writes the factory's access paths, which are at most the release store's read-only pull token, to `state/iam/TENANT.json`, and the cell keeps only that grant (`custody.retain`). The monthly `access` job checks the list, writes a **signed access statement** (`state/access/statement-DATE.json`: every path, the retained grants, anything unlisted) and records its SHA-256 in an `access` audit row, which the next hourly anchor signs into the write-once store, so the statement can't be changed afterwards without the chain showing it. An unlisted path fails the job with a Sev-1 `access_path` alert. Shell and Docker access to the machine itself is the company's own and is outside the cell; the runbook says to keep it to the company's administrators.

The runbook, [self-hosted install](../docs/runbooks/selfhost.md), walks through install, adding people, upgrades, backup and restore, and what to check after each.

**A model on the box** (SB-1710). With the compose profile `model`, the cell answers and embeds with models on its own machine, and no question leaves it. Two containers run llama.cpp's server (MIT licence) from one image pinned by digest: `model`, an instruct model for answers, and `model-embed`, an embedding model, each with its weights read-only from the `models` volume, as the service user on a read-only root, on the `data` network only, with no route out, no published port and no secret. The cell uses them as `model.provider: openaicompat` with `model.base: http://model:8080/v1` and `embed.provider: openaicompat` with `embed.base: http://model-embed:8080/v1` (sb-cell(5)); the model's price is 0, so the budget guard only counts. `make selfhost-model` (`tools/selfhost/model.sh`) tries it: given `SB_MODEL_GGUF` and `SB_EMBED_GGUF`, the weights files the operator has fetched, it records their SHA-256, resolves the image's digest, starts the profile, runs the eval gate's suites against that exact model through a loopback-only port that is removed afterwards, and switches the cell to it **only if the gate passes**; a model that fails the gate is never configured. It writes the model id and the weights' hashes to `build/selfhost/model.json` for the release bundle (`tools/release/bundle --model-weights`, sb-release(7)). `make selfhost-bundle` carries the weights files when `SB_MODEL_GGUF` and `SB_EMBED_GGUF` are set, so an air-gapped site never downloads them. **Hardware and latency:** a cell without the profile runs on 4 vCPU and 16 GB as before; with a 7 to 8 billion parameter model quantised to 4 bits it needs about 8 vCPU and 32 GB, or a small GPU. On CPU an answer takes tens of seconds: measured on Docker Desktop with 16 cores and 23 GB, Qwen2.5-7B-Instruct at Q4_K_M with nomic-embed-text-v1.5 answered in 7 to 32 seconds (median 15) over five asks, and the eval gate took 32 minutes (`make selfhost-smoke` prints the latency it measures). NFR-01's 6-second p95 is met only with a GPU; search (sb-ask(3)) stays the fast path. Changing to the profile's embedder rebuilds the index (sb-ingest(7)).

**The demo.** `make selfhost` with no configuration is a demo of the same layout on a laptop: the example tenant, the example share (`examples/share`, sb-connector(3)) to answer from, the bundled sign-in with one user, `ann@acme.example`, whose password it makes and leaves in `build/selfhost/demo-password`, Caddy's own CA, a stand-in off-box directory (`build/selfhost/.offsite`, a dot directory so Go's tools skip it, on the same disk, which a real install refuses to accept quietly), and a **demo release** of the machine's own images signed with a key made for the machine. The cell is at `https://cell.localhost:8443` (the browser warns about Caddy's certificate until it is trusted, `caddy trust` in the proxy container). Run `make selfhost-update` once to move the services onto the admitted digests. `SB_IDP=dev make selfhost` uses the development IdP instead, which signs in with no password and so refuses any address but loopback. **`make selfhost-smoke`** checks the demo from outside: the services' users and read-only roots; the UI over TLS; a wrong password refused and the right one signed in; answers that cite the share and never a file the asker can't read or one with no ACL file; the database network's missing route out; each vault's owner and mode; anchors in the write-once store and a refused delete; tonight's backup on the box, in the write-once store and off the box; the access statement; `sb tenant verify --cell` green; no Docker socket in any container; each service on its admitted digest; and a release signed with another key changing nothing. `make cell` (sb-runtime(7)) stays the flat, no-container way to run the same binaries for development.

## LAWS

[LAW-02](sb-laws.7.md#law-02): one machine, one tenant's cell. [LAW-06](sb-laws.7.md#law-06): the cell runs only a release it admitted, by digest, and an unadmitted one changes nothing. [LAW-11](sb-laws.7.md#law-11): anchors and the backup's write-once copy can't be overwritten or deleted for seven years. [LAW-07](sb-laws.7.md#law-07): each service reads only its own secrets, at call time, from files nobody else can read. [LAW-16](sb-laws.7.md#law-16): the cell runs its applied manifest, kept in its own state. [LAW-17](sb-laws.7.md#law-17): the factory's only path to the cell is listed, checked monthly and stated in a signed statement. [LAW-18](sb-laws.7.md#law-18): every key is made on the box and held there, by the company.

## EXAMPLES

```text
$ make selfhost
selfhost: acme-logistics is running at https://cell.localhost:8443
selfhost: sign in at https://cell.localhost:8443/login as ann@acme.example with the password in build/selfhost/demo-password
$ make selfhost-update
selfhost-update: running rel_2026-11-21_96f9cb on images pinned by digest

$ sudo tools/selfhost/install.sh --check /etc/second-brain/install.env
install: the machine and /etc/second-brain/install.env look right; nothing changed
$ sudo tools/selfhost/install.sh /etc/second-brain/install.env
install: acme-logistics is running at https://brain.acme.example
```

## SEE ALSO

[sb-idp(1)](sb-idp.1.md), [sb-cell(5)](sb-cell.5.md), [sb-audit(7)](sb-audit.7.md), [sb-connector(3)](sb-connector.3.md), [sb-runtime(7)](sb-runtime.7.md), [sb-release(7)](sb-release.7.md), [sb-identity(7)](sb-identity.7.md)
