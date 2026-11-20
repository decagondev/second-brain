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

### Data, exactly (SB-902)

Cloud SQL for PostgreSQL 17 with `pgvector` has **private IP only**. Services reach it through Direct VPC egress on the cell's own network, and never through a public address or the Cloud SQL proxy's public path. It is encrypted with a **customer-managed key (CMEK)**, a key in the cell's own key ring (`sb-TENANT/cell-TENANT-sql`), which the root key's custodian controls. Only this project's Cloud SQL service agent may use that key. An existing instance that isn't private-IP on the cell's network, or isn't on that key, is refused, never reused. The `pgvector` flag is set when the instance is created. Destroying that key version makes the database unreadable, which is how a cell is crypto-shredded at off-boarding.

A **VPC Service Controls perimeter** surrounds the cell's project. It covers Cloud SQL, Cloud Storage, Secret Manager, Cloud KMS and Pub/Sub, so data can't be copied out by API to a project outside it, even with valid credentials. The only identities allowed through are the cell's own service accounts, plus the factory's `updater` grant for the release store, which is read-only and outside the perimeter. The tenant's organisation admin creates it with `deploy/gcp/bootstrap-perimeter.sh`. Because the perimeter is a second fence, an IAM grant made by mistake from one cell to another still can't reach the other cell's data.

### Schedules, exactly (SB-904)

Every timed job runs from **Cloud Scheduler**, which publishes to the cell's Pub/Sub topic, and the `jobs` Cloud Run job does the work. The schedule is defined once in code (`internal/core/schedule`) and created by the gcp adapter. Times are UTC.

| Job | When | Does | Law |
|---|---|---|---|
| `sync` | every minute (Cloud Scheduler's finest; the jitter is in the job) | incremental sync of every connector (sb-ingest(7), Sync) | LAW-13 |
| `anchor` | hourly, at minute 5 | sign the latest audit hash and write it to the WORM bucket | LAW-11 |
| `retention` | daily at 03:00 | the retention sweep (sb-tenant(5)) | |
| `backup` | daily at 02:00 | the 3-2-1 backup (sb-audit(7)) | LAW-11 |
| `report` | Mondays at 07:00 | the weekly leadership report (sb-observability(7)) | |
| `access` | the 1st of each month at 04:00 | the LAW-17 access-path check, for a tenant-custody cell (sb-factory(7)) | LAW-17 |

Each run records its finish time. A **watchdog**, run by `jobs` every 5 minutes, compares each job's last finish with its schedule. A job that hasn't finished within twice its interval (for `report` and `access`, within a day of its time) is **overdue**: it sets `sb_job_overdue{job}` to 1 and sends a `job_missed` alert. The alert is Sev-1 for `anchor`, `backup` and `access`, because those carry a law, and Sev-2 otherwise. A missed run is never a silent skip. If a job's last finish can't be read, it counts as overdue.

**The WORM bucket.** Anchors and the off-site backup copy go to a Cloud Storage bucket with a **locked** retention policy of seven years (ADR-0012). An object can't be overwritten, every write is conditional on there being no object of that name, and it can't be deleted until its retention ends, by anyone. The gcp adapter refuses a bucket whose policy is unlocked or shorter. The bucket is made, and its policy locked, at bootstrap by the project owner, because locking a retention policy can't be undone. The cell itself only gets `storage.buckets.get` and object create, get and list (`gcp.BucketPermissions`), and never delete.

### Endpoints, exactly (SB-905)

The `api` service serves two endpoints beside the product API. Neither ever returns content, prompts, answers or secrets (LAW-07):

| Endpoint | Caller | Returns |
|---|---|---|
| `GET /metrics` | Managed Service for Prometheus, through the cell's own scrape identity | the Prometheus text format of `obs.Metrics` (sb-observability(7)): ids, codes and numbers only. Labels obey the telemetry rule, so text can't reach a label |
| `GET /v1/verifier/attestation` | the control plane | the running-build attestation, signed by the cell (sb-release(7), Attestation): tenant, release, image digests, sequence number |
| `POST /v1/verifier/verify` | the control plane, for `sb tenant verify --cell` | the verify report: one line per law with `pass`, `fail` or `unknown` and counts (sb-tenant(1)). The body is `{ "laws": ["LAW-01"], "sample": 50, "seed": 7 }` |

Both verifier endpoints require the control plane's identity. Anyone else gets `EAUTH` or `EPERM`, and an unknown law is `EINVAL`. The attestation reports what the cell's services **actually run**, read from Cloud Run at the moment of the call, never what the updater meant to deploy.

**Alerts.** The `jobs` runner evaluates the Sev-1 law rules (sb-observability(7), Law alerts) over each scrape, and sends every firing alert through the alert webhook adapter, with the law, the tenant and the runbook, never content.

## LAWS

[LAW-02](sb-laws.7.md#law-02): every resource and role is in the cell's own project. [LAW-06](sb-laws.7.md#law-06): only admitted, attested images run. [LAW-18](sb-laws.7.md#law-18): the database key is in the cell's key ring.

## EXAMPLES

```text
$ gcloud run services describe api --project gamma-prod-7731 --format='value(spec.template.spec.containers[0].image)'
europe-west1-docker.pkg.dev/sb-factory/releases/api@sha256:4f1c…
```

## SEE ALSO

[sb-tenancy(7)](sb-tenancy.7.md), [sb-release(7)](sb-release.7.md), [sb-factory(7)](sb-factory.7.md), [ADR-0021](../docs/adr/ADR-0021-gcp-cell-runtime.md)
