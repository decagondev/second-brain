---
title: "sb-tenancy(7)"
man: "7"
summary: Cells, isolation and regions.
---

# sb-tenancy(7)

## NAME

sb-tenancy — what a tenant cell is and what it shares with nobody

## SYNOPSIS

```text
one tenant = one cell = { database, index schema, vault namespace, KMS key, queue, region }
```

## DESCRIPTION

Every company is a **tenant**, and every tenant is served by exactly one **cell**, stamped from the same template (ADR-0001).

| In the cell (yours alone) | In the control plane (shared, no content) |
|---|---|
| Postgres database and index schema | Tenant registry: name → cell endpoint, region |
| Vault namespace with connector credentials | Release controller and signed images |
| KMS key encrypting data, backups and exports | Kill-switch flags |
| Queue and ingest workers | Attestation results |
| Audit store | |

### Cell anatomy

The template (`internal/core/cell`) declares every cell the same way. It takes only `tenant` and `region` from the manifest; nothing else changes the shape of a cell.

| Resource | Name | Notes |
|---|---|---|
| `database` | `sb_TENANT` (hyphens become underscores) | Postgres, this cell's data only |
| `index-schema` | `idx` | Inside the cell database; derived, rebuilt from sources (LAW-13) |
| `vault-namespace` | `tenants/TENANT` | Connector credentials |
| `root-key` | `cell-TENANT-root` | Held by the `Keys` port. It wraps per-purpose data keys (envelope encryption), so the root can be re-wrapped at hand-off without re-encrypting data (LAW-18) |
| `queue` | `TENANT-jobs` | Ingest and action jobs |

Every resource carries its cell's `tenant` and `region`. The provider id of each resource is unique to that cell: stamping two cells never returns a shared id, in the same account or in different ones. Stamping is idempotent: stamping a cell again finds what exists and creates only what's missing.

The template never names a cloud account. The `Placement` port stamps resources into the account named by the cell's custody record. Each cloud is an adapter behind `Placement` and `Keys` ([ADR-0018](../docs/adr/ADR-0018-cloud-neutral-cells-behind-ports.md)). The `local` adapter (development and tests) and the `gcp` adapter (below) exist; AWS and Azure adapters come later. If a port can't say whether a resource exists, that is an error, never "absent".

### Placement on GCP, exactly (SB-703)

The `gcp` adapter (`internal/adapters/placement/gcp`) implements `Placement` and `Keys` in one GCP project, for the factory's own project or a tenant's (`tenant` custody, sb-factory(7)). It calls Google's REST APIs with the access token of a **deploy service account** and nothing else. It is the same template, with no tenant-specific code: only the project and the region differ.

| Template resource | GCP resource |
|---|---|
| `database` | a Cloud SQL for PostgreSQL 17 instance `sb-TENANT` (private IP, CMEK `cell-TENANT-sql`, sb-runtime(7)), and the database `sb_TENANT` in it |
| `index-schema` | the schema `idx` in that database, made by the cell's migrations. The adapter ensures the database exists and returns the schema's id |
| `vault-namespace` | the Secret Manager prefix `sb-TENANT-`, marked by a secret `sb-TENANT-namespace`, with replication only in the cell's region |
| `root-key` | the Cloud KMS key `cell-TENANT-root` (symmetric, encrypt and decrypt) in the key ring `sb-TENANT`. `Wrap` and `Unwrap` are KMS `encrypt` and `decrypt`, with the key's name as additional authenticated data, so the key's material never leaves KMS |
| `queue` | the Pub/Sub topic `TENANT-jobs`, with message storage limited to the cell's region |

The manifest's region maps to one GCP location: `eu` is `europe-west1`, `us` is `us-central1`. Any other region is an error. Every resource is labelled `sb-tenant` and `sb-kind`, and has its location. A resource that already exists with a different tenant, kind or location is an error, never reused. Only an HTTP 404 means a resource is absent. Any other failure, a 403 included, is an error, so the adapter never creates a second copy of something it couldn't read. Long-running operations are polled until they finish or the context ends.

What runs in the project, and how it updates itself, is in [sb-runtime(7)](sb-runtime.7.md).

**The deploy role.** The tenant creates it in their own project with `deploy/gcp/bootstrap-deploy-role.sh PROJECT FACTORY_PRINCIPAL`. The script creates a custom role, `sbCellDeployer`, whose permissions are exactly those the adapter uses (`gcp.DeployPermissions`, which a test compares with the script). It creates a service account, `sb-cell-deployer`, with that role on the project and nothing else, and lets only the factory's principal impersonate it. The role can't read secret values, decrypt anything other than through the cell's own key, or change IAM. At hand-off the factory loses this impersonation grant, unless the hand-off record retains it (sb-factory(7)).



The control plane's tenant registry holds one **cell record** per tenant: `tenant`, `region`, the cell `endpoint`, the `hosts` that reach it (by default `TENANT.` plus the control plane's base domain), and the custody record (`custody`, `placement`, `key_custodian`). `sb tenant apply` writes the record when it creates a cell. v1 always writes `factory`, `factory-account` and `factory`. The registry also holds each tenant's applied resources (sb-tenant(1)). In production it is Postgres (ADR-0017); `file:///DIR` keeps it in `DIR/cells` and `DIR/tenants` for development.

The edge router serves every request in the same order:

1. It looks up the request's host in the registry. An unknown host is refused with `EPERM`: unknown means refuse. If the registry can't answer, the request fails with `EUNAVAIL`.
2. It verifies the token. A missing or invalid token, or one with no tenant claim, is `EAUTH`.
3. If the token's tenant isn't the host's tenant, the request is refused with `ETENANT`, and it never reaches any cell.
4. Otherwise it forwards the request to that cell's endpoint and nowhere else.

No code anywhere names a company: routing is data in the registry. A cell knows its control plane only through configuration, `SB_CONTROL_PLANE` (an https URL) and `SB_IDENTITY_CERT` (a certificate path), both required with no defaults, so it can be re-pointed, for example at hand-off ([sb-factory(7)](sb-factory.7.md)).

### Custody

A company can divide its cell into department **spaces** that only narrow access, with the cell staying the tenant boundary ([sb-spaces(7)](sb-spaces.7.md), ADR-0020).

Every cell also has a custody record (`factory`, `tenant` or `detached`) that says which cloud account it lives in and who holds its root key. In v1, every cell is `factory` custody. Custody can later be handed to the company without re-onboarding ([sb-factory(7)](sb-factory.7.md)).

### Region pin

`region` in the manifest is immutable. The supported regions are `eu` and `us` (the working answer to Q-04). Data, backups, restores and model calls stay in that region, where the model provider offers it. Moving region means creating a new tenant and re-ingesting, because the index is derived.

### Scope in code

Inside a cell, every operation carries a `tenancy.Scope` that only the auth middleware can create. There is no API to construct one for another tenant, and no "all tenants" query exists anywhere in cell code.

- A `Scope` has unexported fields (tenant, user, groups, and from SB-1002 the space). The only constructors are `Issuer.Issue` and `Issuer.IssueIn`, which adds a space the middleware has already resolved; `Scope.Space()` is the space's id, or `company` for a company-wide request (sb-spaces(7)). A cell makes exactly one `Issuer`, for its own tenant, at start-up, and hands it only to the auth middleware. A CI check fails if `tenancy.NewIssuer` is called anywhere else outside tests.
- `Issue` refuses a session whose tenant isn't the issuer's tenant. The zero `Scope` is invalid, and every scoped method refuses it, so a forgotten scope is an error, never "no filter".
- Every port method that reads or writes tenant data takes a `Scope` first and uses only `Scope.Tenant()` to choose rows. A query for another tenant's ids returns nothing, exactly as if they didn't exist.
- The middleware puts the scope in the request context. Handlers read it with `tenancy.FromContext`; a request without one is `EAUTH`.

### Per-space data keys (SB-1003)

Each space (sb-spaces(7)) has its own 256-bit data key, the wrap purpose `space/spc_NAME` in the cell's wrap inventory (sb-factory(7), Key wraps). Like every data key it is stored only wrapped by the cell's root key, so a space adds no key a custodian doesn't hold. `sb tenant apply` makes a space's key when it first creates the space, before any content can be written to it.

- **Sealing.** Content held for a space, its chunks and its traces, is sealed with AES-256-GCM under the space's key. The additional authenticated data is `TENANT/spc_NAME`, so a sealed record copied into another space or another cell doesn't open. Every record carries a fresh random nonce.
- **Crypto-shredding.** When the manifest drops a space, apply deletes the space's row and then every wrap of its key, for every custodian. From that moment nothing the space sealed can be opened, in the database or in any backup, and opening it is an error, never an empty result. Every other space's key, the audit signing key and the root key are untouched.
- **Only space keys shred.** Shredding refuses any purpose that isn't `space/…`; the audit signing key and the root key are never removed this way (LAW-18).
- **Hand-off.** Re-wrap covers every purpose, so space keys move to the tenant's custodian with the rest, and destroying the factory's wraps still needs the tenant's to unwrap every key now (LAW-18).

### Off-boarding

Off-boarding is a maintainer procedure, not a manifest field, and needs two approvals. It exports the audit log to the tenant, deletes the cell, and finally destroys the KMS key, which crypto-shreds anything left in backups.

## LAWS

[LAW-02](sb-laws.7.md#law-02), [LAW-13](sb-laws.7.md#law-13).

## EXAMPLES

```text
$ sb tenant apply acme.yaml --plan
+ cell acme-logistics (region eu): postgres, index schema, vault ns, kms key, queue
```

## SEE ALSO

[sb-tenant(5)](sb-tenant.5.md), [sb-factory(7)](sb-factory.7.md), [sb-identity(7)](sb-identity.7.md), [05 Storage and index](../docs/architecture/05-storage-and-index.md)
