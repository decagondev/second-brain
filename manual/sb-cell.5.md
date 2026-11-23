---
title: "sb-cell(5)"
man: "5"
summary: A cell's own configuration, which builds every part of the running cell (SB-1501).
---

# sb-cell(5)

## NAME

sb-cell — the configuration of one running cell

## SYNOPSIS

```json
{
  "tenant": "acme-logistics",
  "manifest": "/etc/sb/acme-logistics.yaml",
  "state": "/var/lib/sb",
  "vault": "/var/lib/sb/vault",
  "policies": "/etc/sb/policy",
  "listen": { "api": "127.0.0.1:8080" },
  "base_url": "https://acme-logistics.sb.localhost",
  "oidc": { "client_id": "sb-acme-logistics" },
  "model": { "provider": "extractive" },
  "durable": false
}
```

## DESCRIPTION

The manifest (sb-tenant(5)) says what a tenant's brain is; this file says how one cell runs it: where its state lives, where its secrets are, what it listens on. The cell's services (`sb-api`, `sb-ingest` and `sb-jobs`) each read it with `--config FILE` and build every part of the cell from it in one place, `internal/cell/wire`, so no part is built twice or differently in two services.

A cell refuses to start, with the reason and nothing served, if:

- the file has an unknown field, a missing required field, or a value of the wrong type;
- the manifest doesn't pass `sb tenant check`, or its `tenant` isn't this file's `tenant`;
- the policies don't load, the state or vault directory can't be opened, or the identity provider can't be reached (its discovery document and keys);
- a connector kind in the manifest has a vendor no adapter serves (the cell would answer `Unknown` for all of it for ever);
- `durable` is true and any store has only an in-memory adapter today (see Stores).

It never starts with a part switched off.

## FIELDS

| Field | Type | Required | Meaning |
|---|---|---|---|
| `tenant` | string | yes | The tenant this cell serves; must match the manifest |
| `manifest` | path | yes | The applied manifest (sb-tenant(5)) |
| `state` | path | yes | The cell's own state directory: audit log, kill switches, key wraps, cursors, logs, backups |
| `vault` | path | yes | The local vault root (`vault://tenants/TENANT/…` paths, sb-tenant(5), Secrets) |
| `policies` | path | yes | The policy directory: `platform/*.cedar` and the tenant's own |
| `listen.api` | host:port | yes | Where `sb-api` listens |
| `base_url` | URL | yes | The cell's public address, for sign-in redirects |
| `oidc.client_id` | string | yes | The cell's OIDC client id at the tenant's IdP; its secret is `vault://tenants/TENANT/oidc` |
| `oidc.issuer` | URL | no | The issuer, when it isn't the manifest's `idp` written as `https://` (a local test IdP) |
| `web` | path | no | The built web UI (`web/dist`); without it, `GET /` is not served |
| `model.provider` | string | yes | `extractive` (a local model that quotes its sources, for development and tests), or a gateway provider of ADR-0019 (`openaicompat`, `anthropic`, `groq`, `openrouter`) with its key at `vault://tenants/TENANT/model` |
| `model.base` | URL | for a gateway provider | The provider's API |
| `model.id` | string | for a gateway provider | The model id |
| `model.price.in_per_m`, `model.price.out_per_m` | number | for a gateway provider | USD per million input and output tokens, until the release bundle carries the price table: the budget guard (`budget_usd_month`), the weekly report and the cost statement use it. `extractive` costs nothing |
| `embed.provider` | string | no, `hashing` | `hashing` (the local 64-dimension embedder), or `openaicompat`: `POST {embed.base}/embeddings` with `embed.model`, no key, returning `embed.dim` numbers a text (SB-1710). A different embedder than the one the index was built with rebuilds the index (sb-ingest(7)) |
| `embed.base`, `embed.model`, `embed.dim` | URL, string, int | with `openaicompat` | The embeddings endpoint (on a self-hosted cell `http://model-embed:8080/v1`, on the `data` network), the model's id, and its vector size (1 to 4096) |
| `connectors.KIND` | object | no | Overrides for one connector: `base` (the vendor API's address, for a sandbox or a recorded fake). For `docs: folder` it is required and names the directory, `file:///DIR`; the connector keeps its listings under `state/connectors/KIND/` (sb-connector(3), SB-1606) |
| `chat.team_id` | string | when `chat: slack` | The tenant's Slack workspace id: the chat app (sb-ask(3)) answers only requests from it, and `POST /connectors/slack/events` takes its events |
| `alerts.webhook` | https URL | no | Where alerts are posted as JSON (sb-observability(7)); every alert is also logged and counted in `sb_alerts_total` |
| `alerts.report_webhook` | https URL | no | Where the weekly report is posted (sb-observability(7), Weekly leadership report); without it, `alerts.webhook` (SB-1608) |
| `backups` | list | for the `backup` job | The nightly backup's copies, each `{dir, media, offsite}` (with `seal: "sb-recovery-pub:…"`, every object sealed to that recovery key before it is written there, as `NAME.sealed`; SB-1901) or `{worm: true}` for the `worm` store: at least three, on two kinds of media, one off-site (sb-audit(7)). Without them the `backup` job fails, and its missed run is a Sev-1 |
| `worm.endpoint`, `worm.bucket` | URL, string | no | The write-once store for anchors and the backup's `worm` copy: an S3 API with compliance-mode object lock of at least seven years (sb-selfhost(7), Write-once storage), its key pair at `vault://tenants/TENANT/worm` as `{"access_key", "secret_key"}`. The cell checks the lock before each write and refuses a bucket without it, in governance mode, or shorter. Without `worm`, anchors go to `state/worm/` (development only) |
| `worm.region`, `worm.offsite` | string, bool | no | The signing region (default `us-east-1`), and whether the store is off this machine, so the backup's `worm` copy counts as off-site |
| `custody.tenant`, `custody.retain` | bool, list | no | A tenant-custody cell (sb-factory(7)): the monthly `access` job checks every factory access path against `retain`, the grants the hand-off record keeps, writes a statement of what it found to `state/access/statement-DATE.json` and its SHA-256 to an `access` audit row, so the next anchor signs it (SB-1605). A self-hosted cell is in this custody from its install (sb-selfhost(7)) |
| `release.store` | URL | for `sb-updater` | The release store: `file:///DIR`, or the factory's `https://` store read with `vault://tenants/TENANT/release-pull` |
| `release.registry` | string | with `release` | Where the bundle's images live |
| `release.trusted`, `release.cosign_identity` | string | exactly one, with `release` | The one release identity the cell trusts: an Ed25519 key (`ed25519:BASE64`), or the release workflow's cosign identity |
| `release.deploy.kind`, `.project`, `.location` | string | with `release` | `local`, or `gcp` with the cell's project and region |
| `database` | DSN | for `durable` | The cell's own Postgres (with `pgvector`), e.g. `postgres:///sb_acme?host=/var/run/postgresql`, or `vault://tenants/TENANT/database` to read that DSN, password and all, from the vault at start (sb-selfhost(7)): every record store, the index, the spaces registry and the audit log live there, and each service creates its tables at start, one at a time (SB-1508). A database on another machine is reached with `sslmode=verify-full`, checked against the system's roots and `/etc/sb/ca` (sb-selfhost(7), A database on another machine; SB-1807) |
| `durable` | bool | no, `false` | `true` refuses to start without a `database`, so no store is in memory |

### Stores

| Store | Adapter |
|---|---|
| audit log, kill switches, key wraps, sync cursors, logs, job finishes (`runs/`), the usage ledger (`usage/`), anchors (`worm/`, write-once), weekly reports (`reports/`), what each service runs (`deploy/`) | files under `state` |
| directory and sessions (a session is kept by a hash of its id, never the cookie value), traces and conversations, actions and receipts, spaces registry, index, chat events inbox, proposals, access reviews, legal holds, and the audit log | the `database` when it is set (`internal/adapters/postgres`); without one, in memory (the audit log in files under `state`), for development and tests only |

### Connectors

Each kind in the manifest's `connectors` is served by the vendor's adapter: `docs: gdrive`, `chat: slack`, `tickets: linear`, `crm: hubspot`, `code: github`, `hris: bamboohr`, `accounting: quickbooks`. Credentials are read from the vault at each call (sb-connector(3)). Any other vendor stops the cell from starting.

### One process or several

`sb-ingest` and `sb-jobs` share the cell's stores with `sb-api`. While any store is in memory there is nothing to share, so they refuse to start on their own and run inside the api instead: `sb-api --config FILE --with ingest,jobs` (or `--with ingest`, `--with jobs`). With durable stores (SB-1508) each runs as its own service, as on GCP.

### Environment

| Variable | Meaning |
|---|---|
| `HTTPS_PROXY`, `NO_PROXY` | The outbound proxy for `https://` calls (connectors, the model gateway, the release store, the company's IdP) and the hosts that skip it, read by Go's HTTP clients; plain `http://` calls inside the layout never use one |
| `SB_HTTPS_PROXY_FILE` | A file holding the proxy's URL, credentials and all (SB-1806): read once at start, before any HTTP client, and used as `HTTPS_PROXY`, so the credentials are never in the process's environment as given. It must be an `http://` or `https://` host and port; setting it with `HTTPS_PROXY` stops the service from starting |
| `SSL_CERT_DIR` | The roots the cell trusts for TLS, the proxy's CA and a private database CA among them on a self-hosted cell |

## EXIT STATUS

`sb-api` exits 0 after a clean shutdown (SIGTERM or SIGINT: it stops taking requests and finishes those in flight), and 1 with the reason if it can't start.

## LAWS

[LAW-02](sb-laws.7.md#law-02): one file, one tenant, one cell. [LAW-03](sb-laws.7.md#law-03): nothing that can't be built is skipped. [LAW-16](sb-laws.7.md#law-16): the cell runs its manifest, nothing else.

## SEE ALSO

[sb-runtime(7)](sb-runtime.7.md), [sb-tenant(5)](sb-tenant.5.md), [sb-api(3)](sb-api.3.md), [sb-connector(3)](sb-connector.3.md)
