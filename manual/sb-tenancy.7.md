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

The template never names a cloud account. The `Placement` port stamps resources into the account named by the cell's custody record. Each cloud is an adapter behind `Placement` and `Keys` ([ADR-0018](../docs/adr/ADR-0018-cloud-neutral-cells-behind-ports.md)). The `local` adapter (development and tests) exists now; AWS, GCP and Azure adapters come later. If a port can't say whether a resource exists, that is an error, never "absent".

### Registry and routing

The control plane's tenant registry holds one **cell record** per tenant: `tenant`, `region`, the cell `endpoint`, the `hosts` that reach it (by default `TENANT.` plus the control plane's base domain), and the custody record (`custody`, `placement`, `key_custodian`). `sb tenant apply` writes the record when it creates a cell. v1 always writes `factory`, `factory-account` and `factory`. The registry also holds each tenant's applied resources (sb-tenant(1)). In production it is Postgres (ADR-0017); `file:///DIR` keeps it in `DIR/cells` and `DIR/tenants` for development.

The edge router serves every request in the same order:

1. It looks up the request's host in the registry. An unknown host is refused with `EPERM`: unknown means refuse. If the registry can't answer, the request fails with `EUNAVAIL`.
2. It verifies the token. A missing or invalid token, or one with no tenant claim, is `EAUTH`.
3. If the token's tenant isn't the host's tenant, the request is refused with `ETENANT`, and it never reaches any cell.
4. Otherwise it forwards the request to that cell's endpoint and nowhere else.

No code anywhere names a company: routing is data in the registry. A cell knows its control plane only through configuration, `SB_CONTROL_PLANE` (an https URL) and `SB_IDENTITY_CERT` (a certificate path), both required with no defaults, so it can be re-pointed, for example at hand-off ([sb-factory(7)](sb-factory.7.md)).

### Custody

Every cell also has a custody record (`factory`, `tenant` or `detached`) that says which cloud account it lives in and who holds its root key. In v1, every cell is `factory` custody. Custody can later be handed to the company without re-onboarding ([sb-factory(7)](sb-factory.7.md)).

### Region pin

`region` in the manifest is immutable. The supported regions are `eu` and `us` (the working answer to Q-04). Data, backups, restores and model calls stay in that region, where the model provider offers it. Moving region means creating a new tenant and re-ingesting, because the index is derived.

### Scope in code

Inside a cell, every operation carries a `tenancy.Scope` that only the auth middleware can create. There is no API to construct one for another tenant, and no "all tenants" query exists anywhere in cell code.

- A `Scope` has unexported fields (tenant, user, groups). The only constructor is `Issuer.Issue`. A cell makes exactly one `Issuer`, for its own tenant, at start-up, and hands it only to the auth middleware. A CI check fails if `tenancy.NewIssuer` is called anywhere else outside tests.
- `Issue` refuses a session whose tenant isn't the issuer's tenant. The zero `Scope` is invalid, and every scoped method refuses it, so a forgotten scope is an error, never "no filter".
- Every port method that reads or writes tenant data takes a `Scope` first and uses only `Scope.Tenant()` to choose rows. A query for another tenant's ids returns nothing, exactly as if they didn't exist.
- The middleware puts the scope in the request context. Handlers read it with `tenancy.FromContext`; a request without one is `EAUTH`.

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
