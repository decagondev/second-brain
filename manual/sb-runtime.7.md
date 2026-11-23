---
title: "sb-runtime(7)"
man: "7"
summary: How a cell runs on GCP - Cloud Run services, the in-cell updater, Binary Authorization, Cloud SQL, schedules and endpoints (W9).
---

# sb-runtime(7)

## NAME

sb-runtime — how one cell runs in its own GCP project, and how it updates itself

## SYNOPSIS

```text
project (one per cell, in the tenant's folder)
  Cloud Run:  api · ingest · web            each with its own service account
  Cloud Run job: updater (pulls, admits, deploys) · jobs (sync, retention, reports, anchors)
  Binary Authorization: only images this cell attested may run
  Cloud SQL (Postgres 17 + pgvector, private IP, CMEK) · Cloud KMS · Secret Manager · Pub/Sub
```

## DESCRIPTION

This page describes the runtime that ADR-0021 chose. It is built in W9 against the recorded GCP fake, and runs in a real project from SB-903. The placement of each resource is in [sb-tenancy(7), Placement on GCP](sb-tenancy.7.md#placement-on-gcp-exactly-sb-703). This page covers what runs, who it runs as, and how a new release gets in.

### One configuration, every service (SB-1501, SB-1502)

Every service of a cell is built from the cell's configuration file (sb-cell(5)) by one composition root, `internal/cell/wire`: `sb-api --config FILE` serves the whole API (sb-api(3)) on `listen.api`, with `GET /healthz` (the process is up) and `GET /readyz` (every part was built and the policies, manifest and identity provider answer). On SIGTERM it stops taking requests and finishes those in flight before it exits. `sb-ingest --config FILE` runs every connector's sync (sb-ingest(7), Sync), and `sb-jobs --config FILE` runs the schedules below on the local clock, or `sb-jobs --config FILE run JOB` runs one job and exits, which is how Cloud Scheduler starts the `jobs` Cloud Run job. A cell whose stores are still in memory runs both inside the api, `sb-api --with ingest,jobs` (sb-cell(5), One process or several). The same binaries run locally and in Cloud Run, and in containers on one machine (sb-selfhost(7)).

**A whole cell on one machine** (SB-1508): `make cell` makes a Postgres database (`SB_CELL_DSN`), writes an example tenant's configuration, vault and manifest under `SB_CELL_DIR` (default `~/.sb-cell`, which must be on a filesystem with Unix permissions, since the vault refuses a secret others can read), and starts a development IdP (`tools/devidp`, loopback only, no passwords; `POST /dev/user?id=usr_…` picks who signs in), `sb-api`, `sb-ingest` and `sb-jobs` against it. `make cell-stop` stops them. `make integration` runs the end-to-end test of a durable cell: ingest, an ask, a draft, an action proposed and approved through to the vendor, a legal hold, `sb tenant verify --cell`, and everything still there after a restart. The same configuration, with `release.deploy.kind: gcp` and the cell's Cloud SQL as `database`, is the GCP path, which waits for a real project (SB-903, SB-908).

### Services, exactly (SB-901)

| Service | Kind | Service account | Its roles, in the cell's project only |
|---|---|---|---|
| `api` | Cloud Run service | `sb-api@PROJECT` | Cloud SQL client, Secret Manager accessor on `sb-TENANT-*`, KMS encrypt and decrypt on the cell key, Pub/Sub publisher |
| `ingest` | Cloud Run service | `sb-ingest@PROJECT` | Cloud SQL client, Secret Manager accessor on `sb-TENANT-connectors-*`, Pub/Sub subscriber |
| `web` | Cloud Run service | `sb-web@PROJECT` | none: it serves the built web UI, which calls `api` |
| `jobs` | Cloud Run job | `sb-jobs@PROJECT` | Cloud SQL client, Storage object creator on the anchor bucket, Pub/Sub publisher |
| `updater` | Cloud Run job | `sb-updater@PROJECT` | Cloud Run admin on this project's services, Binary Authorization attestor for this project |

No service account has a role outside its cell's project, and none has a role on another service's account. `web` has no data access at all. The roles are defined once in code (`gcp.ServiceRoles`), and a test checks that every service has an entry, that no role names a resource outside the cell's project, and that `web` has none.

### Updates are pulled (SB-901, LAW-06)

The factory never deploys into a cell (ADR-0015). Every 15 minutes the cell's own `updater` job:

1. reads which release the cell's channel should run from the release controller (sb-release(7));
2. pulls that bundle and **admits** it with the four checks of sb-release(7), What a cell admits. An unsigned or ungated bundle stops here with `EUNSIGNED` and a Sev-1, and nothing changes;
3. **attests** each image digest the bundle lists, with the cell's own Binary Authorization attestor;
4. deploys each service at that digest, never by tag, as a new revision, and moves traffic to it once its health check passes.

Binary Authorization is a **second guard**. The project's policy admits only images this cell's attestor has signed, so an image pushed by anyone else, even with the deployer's rights, is refused by the platform itself. A tag such as `:latest` is refused before the API is called. Each service records the release id it runs, which the attestation reports (sb-release(7), Attestation).

Exactly (SB-1505): the job is `sb-updater --config FILE` (sb-cell(5), `release`). It reads the channel's release from the store's channel pointer (sb-release(7), Channels), admits it against the one release identity the configuration trusts (an Ed25519 key, or the release workflow's cosign identity), and deploys each service by digest through the configured deployer. The `local` deployer records what each service should run under `state/deploy/` and, like Binary Authorization, refuses an image the cell hasn't attested; the `gcp` deployer is Cloud Run, with the updater's own token from the metadata server, never a key file. A release that isn't admitted changes nothing and sends a Sev-1 `release_unsigned` alert. It needs only files, so it runs on its own whether or not the stores are durable. The verifier's attestation (Endpoints, below) reads what the deployer says each service runs and signs it with the cell's attestation key (`vault://tenants/TENANT/attest-seed` locally); without a deployer it answers `EUNAVAIL`.

On a self-hosted cell (SB-1604) the `local` deployer covers `api`, `ingest`, `jobs` and `updater` (there is no separate `web` service), and nothing inside the cell can start a container: no container holds the Docker socket. The updater only admits and records; a host step run by a systemd timer reads the record, refuses a tag, an image the cell didn't attest or one the machine doesn't hold, starts exactly those digests, checks each container runs its digest, and records that under `state/deploy/running/`. The `local` deployer reports `running/` when it is there, so the attestation says what the containers really run, not what was asked for (sb-selfhost(7), Upgrades).

### Data, exactly (SB-902)

Cloud SQL for PostgreSQL 17 with `pgvector` has **private IP only**. Services reach it through Direct VPC egress on the cell's own network, and never through a public address or the Cloud SQL proxy's public path. It is encrypted with a **customer-managed key (CMEK)**, a key in the cell's own key ring (`sb-TENANT/cell-TENANT-sql`), which the root key's custodian controls. Only this project's Cloud SQL service agent may use that key. An existing instance that isn't private-IP on the cell's network, or isn't on that key, is refused, never reused. The `pgvector` flag is set when the instance is created. Destroying that key version makes the database unreadable, which is how a cell is crypto-shredded at off-boarding.

A **VPC Service Controls perimeter** surrounds the cell's project. It covers Cloud SQL, Cloud Storage, Secret Manager, Cloud KMS and Pub/Sub, so data can't be copied out by API to a project outside it, even with valid credentials. The only identities allowed through are the cell's own service accounts, plus the factory's `updater` grant for the release store, which is read-only and outside the perimeter. The tenant's organisation admin creates it with `deploy/gcp/bootstrap-perimeter.sh`. Because the perimeter is a second fence, an IAM grant made by mistake from one cell to another still can't reach the other cell's data.

### Schedules, exactly (SB-904)

Every timed job runs from **Cloud Scheduler**, which publishes to the cell's Pub/Sub topic, and the `jobs` Cloud Run job does the work. The schedule is defined once in code (`internal/core/schedule`) and created by the gcp adapter. Times are UTC.

| Job | When | Does | Law |
|---|---|---|---|
| `sync` | every minute (Cloud Scheduler's finest; the jitter is in the job) | incremental sync of every connector (sb-ingest(7), Sync) | LAW-13 |
| `anchor` | hourly, at minute 5 | sign the latest audit hash and write it to the WORM bucket | LAW-11 |
| `retention` | daily at 03:00 | the retention sweep (sb-tenant(5)), then counts the index's leftovers for health (sb-admin(3); SB-2905) | |
| `backup` | daily at 02:00 | the 3-2-1 backup (sb-audit(7)) | LAW-11 |
| `report` | Mondays at 07:00 | the weekly leadership report (sb-observability(7)) | |
| `review` | the 1st of January, April, July and October at 05:00 | open the quarter's access review for each restricted space (sb-spaces(7), Access reviews) | LAW-16 |
| `drift` | hourly, at minute 35 | compare what the cell runs with its applied manifest and send a `manifest_drift` alert per difference (sb-tenant(5), Drift) | LAW-16 |
| `notify` | every 5 minutes | tell stewards about new questions and askers about replies (sb-ask(3), Notifications) | |
| `evals` | hourly, at minute 50 | run the tenant's cases from wrong citations on the release the cell runs, once per release, and hold a release that regresses one (sb-evals(7), Cases from wrong citations; SB-2204) | LAW-06 |
| `access` | the 1st of each month at 04:00 | the LAW-17 access-path check, for a tenant-custody cell (sb-factory(7)) | LAW-17 |

Each run records its finish time, and only a run that succeeded records one, so a failing job looks exactly like a missed one. Locally the record is a file per job under `state/runs/`. `anchor` signs the audit log's last row with the cell's anchor key and writes the anchor write-once for seven years (locally under `state/worm/anchors/`); `retention` reads the legal holds and each space's `retention_days` when it runs (sb-tenant(5), Retention; sb-space(5)); `report` writes last week's report to `state/reports/WEEK.md`; `review` opens the quarter's reviews that don't exist yet; and `access` on a factory-custody cell has nothing to check and finishes. A **watchdog**, run by `jobs` every 5 minutes, compares each job's last finish with its schedule. A job that hasn't finished within twice its interval (for `report`, `access` and `review`, within a day of its time) is **overdue**: it sets `sb_job_overdue{job}` to 1 and sends a `job_missed` alert. The alert is Sev-1 for every job that carries a law (`sync`, `anchor`, `backup`, `access`, `review`, `drift` and `evals`) and Sev-2 otherwise. A missed run is never a silent skip. If a job's last finish can't be read, it counts as overdue.

**The WORM bucket.** Anchors and the off-site backup copy go to a Cloud Storage bucket with a **locked** retention policy of seven years (ADR-0012). An object can't be overwritten, every write is conditional on there being no object of that name, and it can't be deleted until its retention ends, by anyone. The gcp adapter refuses a bucket whose policy is unlocked or shorter. The bucket is made, and its policy locked, at bootstrap by the project owner, because locking a retention policy can't be undone. The cell itself only gets `storage.buckets.get` and object create, get and list (`gcp.BucketPermissions`), and never delete. A self-hosted cell uses an S3 store with compliance-mode object lock instead, made and locked at install (sb-selfhost(7), Write-once storage).

### Endpoints, exactly (SB-905)

The `api` service serves two endpoints beside the product API. Neither ever returns content, prompts, answers or secrets (LAW-07):

| Endpoint | Caller | Returns |
|---|---|---|
| `GET /metrics` | Managed Service for Prometheus, through the cell's own scrape identity | the Prometheus text format of `obs.Metrics` (sb-observability(7)): ids, codes and numbers only. Labels obey the telemetry rule, so text can't reach a label |
| `GET /v1/verifier/attestation` | the control plane | the running-build attestation, signed by the cell (sb-release(7), Attestation): tenant, release, image digests, sequence number |
| `POST /v1/verifier/verify` | the control plane, for `sb tenant verify --cell` | the verify report: one line per law with `pass`, `fail` or `unknown` and counts (sb-tenant(1)). The body is `{ "laws": ["LAW-01"], "sample": 50, "seed": 7 }` |

Both verifier endpoints require the control plane's identity. Anyone else gets `EAUTH` or `EPERM`, and an unknown law is `EINVAL`. The attestation reports what the cell's services **actually run**, read from Cloud Run at the moment of the call, never what the updater meant to deploy.

**Alerts.** The `jobs` runner evaluates the Sev-1 law rules (sb-observability(7), Law alerts) over each scrape, and sends every firing alert through the alert webhook adapter, with the law, the tenant and the runbook, never content.

### Web UI hosting, exactly (SB-906)

Each tenant's web UI is a **Firebase Hosting site in that cell's own project**, named `sb-TENANT`. It serves the built `web/` from the release bundle, and its configuration is generated, never written by hand (`internal/core/hosting`):

| Rule | Value |
|---|---|
| `/v1/**` | rewrite to the Cloud Run service `api` in the cell's region, in the same project |
| `**` | the single-page app's `index.html` |
| headers | `Content-Security-Policy: default-src 'self'; frame-ancestors 'none'` (as sb-ask(3), The web UI), `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer` |

Firebase Hosting can only rewrite to a Cloud Run service in its own project, and the generator checks that too. A configuration is refused if it rewrites to any service other than `api`, to another region, to a function, or to any absolute URL, or if the site isn't in the cell's project. So no site can ever send a request to another tenant's cell (LAW-02). A tenant may instead serve the same files from its own host, provided that host sends `/v1/` only to its own cell's `api`.

## LAWS

[LAW-02](sb-laws.7.md#law-02): every resource and role is in the cell's own project. [LAW-06](sb-laws.7.md#law-06): only admitted, attested images run. [LAW-18](sb-laws.7.md#law-18): the database key is in the cell's key ring.

## EXAMPLES

```text
$ gcloud run services describe api --project gamma-prod-7731 --format='value(spec.template.spec.containers[0].image)'
europe-west1-docker.pkg.dev/sb-factory/releases/api@sha256:4f1c…
```

## SEE ALSO

[sb-tenancy(7)](sb-tenancy.7.md), [sb-release(7)](sb-release.7.md), [sb-factory(7)](sb-factory.7.md), [ADR-0021](../docs/adr/ADR-0021-gcp-cell-runtime.md)
