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

### Secrets

Secrets never appear in a manifest, in configuration, in a log line or in an audit row (LAW-07). They live in the cell's vault, and code refers to them only by reference. The only valid references are:

| Reference | Holds |
|---|---|
| `vault://tenants/TENANT/connectors/KIND` | the credential for one connector kind |
| `vault://tenants/TENANT/oidc` | the cell's OIDC client secret |
| `vault://tenants/TENANT/scim` | the IdP's SCIM bearer token |
| `vault://tenants/TENANT/model` | the model gateway's API key (ADR-0019) |
| `vault://tenants/TENANT/canary` | the cell's canary secret; nothing may use it ([sb-observability(7)](sb-observability.7.md#canaries-and-the-secret-scanner)) |
| `vault://tenants/TENANT/connectors/docs-classification` | the Drive label that carries sensitivity, no secret in it (sb-connector(3)); optional |
| `vault://tenants/TENANT/audit/anchor-seed` | the cell's audit anchor key seed, 32 bytes in base64, on a local cell (sb-audit(5), Anchors); on GCP it is a data key wrapped by the cell's KMS key |
| `vault://tenants/TENANT/attest-seed` | the cell's attestation key seed, 32 bytes in base64, on a local cell (sb-release(7), Attestation) |
| `vault://tenants/TENANT/control-plane` | the control plane's identity token, which the cell compares for apply, the live configuration and the verifier (sb-api(3)) |
| `vault://tenants/TENANT/release-pull` | the read token for the factory's release store (sb-release(7), Cells in a tenant's account) |
| `vault://tenants/TENANT/database` | the cell database's DSN, when sb-cell(5) `database` names this path (sb-selfhost(7)) |
| `vault://tenants/TENANT/worm` | the write-once store's S3 key pair, `{"access_key", "secret_key"}`, when sb-cell(5) sets `worm` (sb-selfhost(7), Write-once storage) |

A reference is resolved through the `Secrets` port at the moment of the call, and only for the cell's own tenant: a reference to another tenant's path is refused. Resolved values are never stored in a struct field and never cached past the call. Every value resolved is also registered with the logger's redactor, so it is replaced by `[REDACTED]` anywhere it would appear in a log line: message, attribute, error or nested group. Attributes whose names contain `secret`, `password`, `token` or `key` are redacted whatever their value. For development, `file:///DIR` keeps the vault in `DIR/vault/tenants/…`, one file per secret, readable only by its owner; a file anyone else can read is refused.

## FIELDS

| Field | Type | Required | Default | Rules |
|---|---|---|---|---|
| `tenant` | string | yes | | DNS label, 3–40 chars, `[a-z0-9-]`. **Immutable** |
| `channel` | `stable` \| `testing` | yes | | `version:` and `platform:` fields are rejected (ADR-0004) |
| `idp` | URI | yes | | `oidc://ISSUER-HOST[/path]`, where the host is a DNS name, or `identity-platform://PROJECT/IDP_TENANT` for a company signing in through Identity Platform ([sb-identity(7)](sb-identity.7.md)). SCIM is served at `/scim/v2` on the cell |
| `region` | string | yes | | `eu` or `us`, the supported regions ([sb-tenancy(7)](sb-tenancy.7.md)). **Immutable** |
| `connectors` | map kind→vendor | yes | | Kinds: `docs`, `chat`, `tickets`, `crm`, `code` (SB-1103), `hris` (SB-1104) and `accounting` (SB-1105). At least one. Vendor is `[a-z0-9-]`, e.g. `sharepoint`; `docs: folder` reads a directory, such as a mounted file share (sb-connector(3), SB-1606). Read-only unless named in `workflows` |
| `agents` | list | yes | | `ask` (required), `draft-reply`, and the department agents `incident-summary`, `spec-digest`, `close-checklist` and `runbook-answer` (sb-agents(7)). No duplicates |
| `workflows` | list | no | `[]` | At most **one** in v1. Known workflows: `ticket-update` (needs connector kind `tickets`). Enabled only after the tenant's workflow evals pass |
| `approval` | map workflow→principal | if `workflows` | | Principal is `group:NAME` from the IdP. Every workflow needs one |
| `duties` | map workflow→`{ second: group:NAME, above: N }` | no | none | Segregation of duties ([sb-approvals(7)](sb-approvals.7.md)): actions with an `amount` above `N` (or no amount) need a second approver from `second`. `second` differs from the workflow's `approval` group; `N` is a whole number, at least 0 |
| `retention_days` | int | no | `365` | 30–3650. Applies to traces (with their citations and decision lists), drafts, conversations, feedback and logs, never to audit or the index. See Retention and deletion |
| `budget_usd_month` | int | no | `1000` | At least 1. Model spend cap per calendar month (UTC). Over budget, asks fail with `EBUDGET`, never unmetered ([sb-observability(7)](sb-observability.7.md#budgets-and-rate-limits-sb-609)) |
| `pilot_group` | string | no | | `group:NAME`. If set, only this group may sign in (days 1–14) |
| `questions_to` | string | no | none | `group:NAME`: who answers questions sent from company-wide asks that nothing answered (sb-ask(3), Questions to the stewards; SB-1704). Without it, only a space's questions reach anyone |
| `notes` | bool | no | `false` | Stewards may publish replies as notes into `notes/` of the `docs: folder` share (sb-connector(3); SB-1705). Needs `connectors: { docs: folder }` |
| `exclude` | map kind→list | no | `{}` | Kinds as in `connectors`. Source paths or channels never ingested, e.g. `chat: ["#hr-*"]` |
| `groups` | map `group:NAME`→list of emails | no | none | Only with an `identity-platform://` idp: the steward-managed group list for a company without SCIM ([sb-identity(7)](sb-identity.7.md)). At most 200 groups and 2,000 members in all; each email at most once per group |
| `spaces` | map name→space | no | none | Department spaces ([sb-space(5)](sb-space.5.md), [sb-spaces(7)](sb-spaces.7.md)). Checked and planned from SB-1001 and applied from SB-1008. Without it, the cell is one undivided brain, as today |
| `barriers` | list of group pairs | no | none | Information barriers ([sb-space(5)](sb-space.5.md)). Checked, planned and applied like `spaces` |

## VALIDATION RULES

1. Every key is known, no key appears twice, and types match. `tenant` and `region` must match the existing cell if one exists; `check` has no network, so `sb tenant apply` enforces that part (SB-104).
2. `ask` is in `agents`.
3. Each item in `workflows` has an `approval` entry, and its connector kind is in `connectors`. Each key of `duties` is in `workflows`, has exactly `second` and `above`, and its `second` group isn't the workflow's `approval` group.
4. No field anywhere is named `version`, `platform`, `secret`, `password`, `token` or `key`, alone or as part of a `_`-separated name such as `api_token`.
5. `pilot_group`, `approval` and `duties` principals are `group:` references, never individual users. That way access follows the IdP.
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

The same Go types that drive the checker generate a JSON Schema, published as [`examples/tenants/tenant.schema.json`](../examples/tenants/tenant.schema.json) for editors. A test fails if the published file is out of date.

## EXIT STATUS / ERRORS

`sb tenant check` exits 1 with `EMANIFEST` and one line per error, in the form `file:line:col: field: message`, sorted by position. Errors about a field point at its key; errors about a value point at the value. See [sb-tenant(1)](sb-tenant.1.md).

## MODEL BILLING AND COST STATEMENTS

Exactly (SB-709).

**Whose model account.** The gateway's API key is always the cell's own, at `vault://tenants/TENANT/model`, and it is resolved at the moment of each call, never cached and never shared between cells (ADR-0019). Whoever puts the key there holds the provider contract and receives the provider's invoice. In `factory` custody that is usually the factory. A company that wants its own contract, and every company after hand-off, puts a key from **its own** provider account at that path. The next model call uses it, with no deploy and no manifest change, because the key is read per call. The provider and model still come from the release bundle, so changing the key changes who pays, never which model answers.

**Cost statements.** Each month the cell produces a statement for its tenant, from its own metering (the provider's reported usage, SB-609):

| Line | Meaning |
|---|---|
| `month` | the calendar month, UTC, e.g. `2027-03` |
| `provider`, `model` | from the release bundle that served the calls |
| `tokens_in`, `tokens_out` | reported by the provider, summed over the month |
| `price_in_per_m`, `price_out_per_m` | the release's price table, USD per million tokens |
| `cost_usd` | tokens times price, rounded to the cent |
| `budget_usd_month` | the manifest's budget, for comparison |

A statement holds counts and prices only, never content or user ids, so it can be sent to finance as is. `GET /v1/admin/costs?month=2027-03` returns it (sb-admin(3)) as JSON, or as CSV with `&format=csv`, one row per provider and model. The statement is an estimate at list price: when the tenant holds its own key, the provider's invoice is the bill, and the statement is how to check it. A month that hasn't ended is marked `partial`.

## RETENTION AND DELETION

Exactly (SB-611).

**The nightly job.** At 03:00 UTC each cell deletes everything of its tenant older than `retention_days` from every store of derived data:

| Store | What it holds | Aged by |
|---|---|---|
| `traces` | answers and drafts, with their citations and decision lists | when it was answered |
| `conversations` | the questions of each conversation | the last question |
| `feedback` | ratings and comments | when it was given |
| `logs` | the cell's daily log files | the file's day |

A record exactly `retention_days` old is kept, and one older is deleted. The job refuses to run with a `retention_days` outside 30 to 3650, so a bad value fails rather than deleting more than it should. Each run writes one `kind: retention` audit row (subject `sweep`) with the cutoff and how many records each store deleted, never what they held. If any store fails, the row records it, and the next night tries again. Traces are readable for at most 30 days whatever `retention_days` says (sb-ask(3)); retention only bounds how long they are kept.

**What retention never touches.**

- **Audit** follows its own policy: seven years, WORM (sb-audit(7)).
- **The index** is derived from the sources and ages with them. A document deleted at its source leaves the index within one sync (SB-209), and a rebuild gives the same index (LAW-13). The job has no access to it.

**Legal holds (SB-1207).** A hold placed by a Legal steward (sb-spaces(7), Legal holds) keeps every record in its scope, a space, custodians, or both, out of the sweep for as long as it is active, and refuses subject erasure of its custodians and removal of its space. Each sweep's audit row also counts what holds kept (`held=N`).

**Subject deletion.** `POST /v1/admin/subjects/erase` (sb-admin(3)) erases everything the cell's derived stores hold about one user, whatever its age: their traces and drafts, conversations, feedback, and every log line that names them. It writes a `kind: retention` audit row (subject `erase:USER`, state `requested`) **before** erasing anything, and a second row (state `done`, or `failed` with the store that failed) after. If the first row can't be written, nothing is erased. The audit rows about the user, including their earlier actions and approvals, stay: they are the record, and they hold ids, not content. Content about the person inside the sources (a CRM contact, a document they wrote) is deleted at the source, and the index follows (LAW-13).

**Subject export (SB-1903).** `POST /v1/admin/subjects/export` (sb-admin(3)) gives a data-protection officer a copy of exactly what erase would remove, by the same selection in each store, audited before and after with the copy's SHA-256 and never its text. Export, erase, export gives an empty second export.

**The plan (SB-1905).** `GET /v1/admin/retention/plan?days=N` and `sb tenant retention --plan` show what the sweep would delete, by kind, and what holds keep, from the same selection the sweep uses, deleting nothing.

## DRIFT

Every hour, each cell reports its live configuration as resources in the plan's terms (kind, name, attributes), and the control plane compares it with what was last applied from the manifest (SB-510). Any difference is a `manifest_drift` alert that names the resource and the field, for example `connector tickets: attribute vendor: applied jira, live linear`, or `agent draft-reply: present live, not applied`. Values are configuration, never secrets (secrets aren't in the manifest). Exactly (SB-1506): the cell's `drift` job (sb-runtime(7), Schedules) compares what it runs, with its spaces as the registry holds them, against the manifest file it last applied, and `GET /v1/cell/live` gives the control plane the same resources. A manifest that can't be read counts as a failed run, never as no drift. Drift isn't fixed automatically: `sb tenant apply` with the manifest makes the cell match again, and if the live change was intended, the manifest changes first (LAW-16).

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
