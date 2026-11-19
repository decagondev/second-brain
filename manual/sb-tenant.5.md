---
title: "sb-tenant(5)"
man: "5"
summary: The tenant manifest, the one file a company edits.
---

# sb-tenant(5)

## NAME

sb-tenant — tenant manifest format

## SYNOPSIS

The smallest valid manifest:

```yaml
tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint }
agents: [ask]
```

## DESCRIPTION

A manifest fully describes one tenant. It is YAML, lives in git, and is the single source of truth: the console edits it, `sb tenant apply` enacts it, and drift from it is an alert (LAW-16).

A manifest contains **no secrets and no versions**. Connector credentials are read from the fixed vault path `vault://tenants/<tenant>/connectors/<kind>`, which the operator fills once during onboarding. The release is chosen by `channel`, never pinned.

Unknown fields are errors. There are no per-company extension points.

## FIELDS

| Field | Type | Required | Default | Rules |
|---|---|---|---|---|
| `tenant` | string | yes | | DNS label, 3–40 chars, `[a-z0-9-]`. **Immutable** |
| `channel` | `stable` \| `testing` | yes | | `version:` and `platform:` fields are rejected (ADR-0004) |
| `idp` | URI | yes | | `oidc://ISSUER-HOST[/path]`, where the host is a DNS name. SCIM is served at `/scim/v2` on the cell |
| `region` | string | yes | | `eu` or `us`, the supported regions ([sb-tenancy(7)](sb-tenancy.7.md)). **Immutable** |
| `connectors` | map kind→vendor | yes | | Kinds: `docs`, `chat`, `tickets`, `crm`. At least one. Vendor is `[a-z0-9-]`, e.g. `sharepoint`. Read-only unless named in `workflows` |
| `agents` | list | yes | | `ask` (required), `draft-reply`. No duplicates |
| `workflows` | list | no | `[]` | At most **one** in v1. Known workflows: `ticket-update` (needs connector kind `tickets`). Enabled only after the tenant's workflow evals pass |
| `approval` | map workflow→principal | if `workflows` | | Principal is `group:NAME` from the IdP. Every workflow needs one |
| `retention_days` | int | no | `365` | 30–3650. Applies to traces, citations and drafts, not to audit |
| `budget_usd_month` | int | no | `1000` | At least 1. Model spend cap. Over budget, asks fail with `EBUDGET` |
| `pilot_group` | string | no | | `group:NAME`. If set, only this group may sign in (days 1–14) |
| `exclude` | map kind→list | no | `{}` | Kinds as in `connectors`. Source paths or channels never ingested, e.g. `chat: ["#hr-*"]` |

## VALIDATION RULES

1. Every key is known, no key appears twice, and types match. `tenant` and `region` must match the existing cell if one exists; `check` has no network, so `sb tenant apply` enforces that part (SB-104).
2. `ask` is in `agents`.
3. Each item in `workflows` has an `approval` entry, and its connector kind is in `connectors`.
4. No field anywhere is named `version`, `platform`, `secret`, `password`, `token` or `key`, alone or as part of a `_`-separated name such as `api_token`.
5. `pilot_group` and `approval` principals are `group:` references, never individual users. That way access follows the IdP.
6. `custody`, `placement` and `key_custodian` are rejected: custody is a factory record that changes only through a dual-approved hand-off ([sb-factory(7)](sb-factory.7.md)).

## SYNTAX

A manifest is one YAML document in a strict subset of YAML. The parser keeps the line and column of every key and value, so each error points at the exact spot. A syntax error stops the check at the first one, reported as field `syntax`.

| Allowed | Rejected (error) |
|---|---|
| A block mapping at the top level | A second document (`---`), or a top level that isn't a mapping |
| Nested block mappings and `- item` lists | Tabs in indentation |
| Flow lists `[a, b]` and flow maps `{ k: v }`, on one line | Anchors `&`, aliases `*` and tags `!` |
| Plain, `'single'` and `"double"` quoted scalars | Block scalars (literal and folded) and multi-line scalars |
| `#` comments | Duplicate keys |

The same Go types that drive the checker generate a JSON Schema, published as `examples/tenants/tenant.schema.json` for editors. A test fails if the published file is out of date.

## EXIT STATUS / ERRORS

`sb tenant check` exits 1 with `EMANIFEST` and one line per error, in the form `file:line:col: field: message`, sorted by position. Errors about a field point at its key; errors about a value point at the value. See [sb-tenant(1)](sb-tenant.1.md).

## LAWS

[LAW-07](sb-laws.7.md#law-07) (no secrets in config), [LAW-08](sb-laws.7.md#law-08) (channels, not versions), [LAW-16](sb-laws.7.md#law-16) (the manifest is the truth).

## EXAMPLES

A day-0 pilot:

```yaml run
tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack }
agents: [ask]
pilot_group: group:sb-pilot
exclude: { chat: ["#hr-*", "#legal-*"] }
```

Day 46 onward, with one gated workflow:

```yaml run
tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack, tickets: jira }
agents: [ask, draft-reply]
workflows: [ticket-update]
approval: { ticket-update: group:it-leads }
retention_days: 365
budget_usd_month: 1500
```

## SEE ALSO

[sb-tenant(1)](sb-tenant.1.md), [sb-tenancy(7)](sb-tenancy.7.md), [sb-factory(7)](sb-factory.7.md), [sb-onboarding(7)](sb-onboarding.7.md), [examples/tenants/](../examples/README.md)
