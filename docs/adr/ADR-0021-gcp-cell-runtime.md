---
title: "ADR-0021: Cells run on Cloud Run with Cloud SQL, one project per cell; Firebase only for hosting and identity"
summary: "The GCP cell runtime: Cloud Run, Cloud SQL Postgres 17 with pgvector, KMS, Secret Manager, Pub/Sub and Scheduler; Firebase for hosting and Identity Platform only."
---

# ADR-0021: Cells run on Cloud Run with Cloud SQL, one project per cell; Firebase only for hosting and identity

| Status | Date | Deciders |
|---|---|---|
| Accepted (plan, W9) | 2026-11-19 | Platform lead (the user) |

## Context

W7 built the GCP `Placement` and `Keys` adapter (SB-703) against a recorded fake: Cloud SQL, Cloud KMS, Secret Manager and Pub/Sub. What's left in W0 to W7 is mostly the runtime: something that runs the cell's services, its schedules (sync, retention, weekly report, monthly LAW-17 check, hourly audit anchor), `/metrics` and the verifier endpoint. ADR-0011 rules out Kubernetes in v1. Department spaces (ADR-0020) need nothing new from the cloud, but they do need a cell that really runs.

## Decision

- **Compute:** Cloud Run services per cell (`api`, `ingest`, `web` origin), each with its own service account. Jobs run as Cloud Run jobs.
- **Data:** Cloud SQL for PostgreSQL 17 with `pgvector`, private IP only, encrypted with a customer-managed key (CMEK) from the cell's Cloud KMS key ring. Cloud Run reaches it through Direct VPC egress.
- **Keys and secrets:** Cloud KMS (root key and space data keys, ADR-0020), Secret Manager (vault references, SB-205).
- **Events and schedules:** Pub/Sub for the queue; Cloud Scheduler for every timed job.
- **Audit anchors:** a Cloud Storage bucket with a locked retention policy as the WORM store (ADR-0012).
- **Layout:** one project per cell, one folder per tenant, a factory folder for the control plane. Placement already takes the project as a parameter (ADR-0015, ADR-0018).
- **Perimeter:** a VPC Service Controls perimeter around each cell project.
- **Builds:** Binary Authorization admits only images whose signature the release controller verified (LAW-06).
- **Firebase where it fits:** Firebase Hosting serves the web UI per tenant, with rewrites to the cell's Cloud Run `api`. Identity Platform (Firebase Authentication's enterprise tier) is an optional `oidc` adapter for companies with no OIDC IdP: one Identity Platform tenant per company, federating SAML or OIDC.
- **Not Firestore.** Postgres stays the single store for the index, audit and approvals, so the hash chain, transactions and `pgvector` stay in one place.
- **IaC:** the adapter keeps calling Google's REST APIs from Go (as SB-703 does); `deploy/gcp/` scripts cover bootstrap only. No second template language (ADR-0018).

## Alternatives rejected

| Option | Why not |
|---|---|
| GKE | ADR-0011; a cluster per region to run for a handful of stateless services |
| Firestore for content or audit | Two stores, no SQL transactions across audit and approvals, and no `pgvector` |
| AlloyDB | Better at scale, but a higher floor cost per cell; revisit with an ADR when a tenant needs it |
| Vertex AI Vector Search | A second index outside the cell's database and key; Postgres keeps LAW-13 rebuilds simple |
| Terraform or OpenTofu as the template | ADR-0018 rejected IaC text as the template; the Go adapter is unit-tested against a fake |

## Consequences

- SB-104 ("cell in under 30 minutes") and other real-cloud items can close in a real project in W9.
- Firebase is an optional edge, not a dependency: a company can use its own CDN or none.
- Cloud Run cold starts affect ask latency; W9 measures p95 with minimum instances set per tenant.
