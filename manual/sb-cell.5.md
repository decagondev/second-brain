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
| `base_url` | URL | yes | The cell's public address, for sign-in redirects and the links in notifications (sb-ask(3), Notifications) |
| `oidc.client_id` | string | yes | The cell's OIDC client id at the tenant's IdP; its secret is `vault://tenants/TENANT/oidc` |
| `oidc.issuer` | URL | no | The issuer, when it isn't the manifest's `idp` written as `https://` (a local test IdP) |
| `web` | path | no | The built web UI (`web/dist`); without it, `GET /` is not served |
| `mcp_groups` | map group → list | no | (SB-5305) Who may use MCP, and which tools: each IdP group named gets the tool families listed, of `search`, `ask` (with `passage`), `agents`, `research`, `skills` (prompts) and `memory` (the person's memory resource and `memory_propose`, SB-5505). A person in none of the groups is refused (`EPERM`), and a tool not granted to any of their groups isn't listed or callable. Without it, everyone gets every family, as before. It needs `mcp: true` |
| `wide_group_size` | integer | no | (SB-5401) A group grant with more members than this shares a document too widely (sb-permissions(7), Shared too widely). Default 500 |
| `mcp` | bool | no | (SB-3801) `true` serves `POST /v1/mcp`, the read-only MCP endpoint other AI tools use with a person's own API token (sb-api(3), MCP). Off by default |
| `model.provider` | string | yes | `extractive` (a local model that quotes its sources, for development and tests), or a gateway provider of ADR-0019 (`openaicompat`, `anthropic`, `groq`, `openrouter`) with its key at `vault://tenants/TENANT/model` |
| `model.base` | URL | for a gateway provider | The provider's API |
| `model.id` | string | for a gateway provider | The model id |
| `model.small.id`, `model.small.price.in_per_m`, `model.small.price.out_per_m` | string, numbers | no | (SB-4405) A second, smaller model of the same provider, base and key: short plain asks go to it (sb-evals(7), Model tiers). Without it, every call uses `model.id` |
| `model.fallback.provider`, `model.fallback.base`, `model.fallback.id`, `model.fallback.price.in_per_m`, `model.fallback.price.out_per_m` | strings, numbers | no | (SB-5601) A second provider and model for the main tier, its key at `vault://tenants/TENANT/model-fallback`. It answers only while the evals job has gated it for the release the cell runs (sb-runtime(7), The fallback model): when the main provider fails, is rate limited or times out, and for 10 minutes after 3 failures in 5 minutes. Without it, or while it isn't gated, a failing main model gives passages only (sb-ask(3), When the model is down) |
| `model.deep.id`, `model.deep.effort`, `model.deep.price.in_per_m`, `model.deep.price.out_per_m` | strings, numbers | no | (SB-5704) The deep tier: a deeper model of the same provider, base and key (`id`), or the main model asked for a higher reasoning effort (`effort`: `low`, `medium` or `high`, sent as `reasoning_effort`; OpenAI-compatible providers only), or both, with its own price. It writes only research syntheses, answers with a table, consultations' syntheses and one retry of a mostly uncited answer, and only while the evals job has gated it (sb-runtime(7), The deep tier) |
| `model.region` | string | no | (SB-5705) The region every model call must stay in, for example `eu`. With it set, no call of any tier, batched or not, goes to a provider that declares another region or none (LAW-33); such a call is refused and an ask shows passages. A remote embedder outside it stops the cell from starting. Empty is anywhere |
| `model.provider_region`, `model.fallback.region`, `embed.region`, `embed.local` | strings, bool | with `model.region` | (SB-5705) Where the main provider (and its small and deep tiers), the fallback and a remote embedder run; `embed.local` is an embedder on the cell's own machine. A local model is in every region. The health report's `model.regions` and the inventory list them |
| `calendar.kind: outlook` | string | no | (SB-5905) Meeting prep, `/v1/me/events` and the leader's week read Outlook calendars through Microsoft Graph (`microsoft`, above) instead of Google's; each person's own events only |
| `model.batch` | bool | no | (SB-5702) `true` says an `openaicompat` provider has OpenAI's file-based Batch API: briefs, routines, rhythm steps and eval probes go through it at half price (sb-runtime(7), Batching). Without it every call runs as before. Refused for any other provider and for a local model |
| `model.price.cached_in_per_m` | number | no | (SB-5605) The price of input tokens the provider read from its prompt cache, per million; 0 or absent is `in_per_m`. Spend, budgets and limits use it |
| `model.price.in_per_m`, `model.price.out_per_m` | number | for a gateway provider | USD per million input and output tokens, until the release bundle carries the price table: the budget guard (`budget_usd_month`), the weekly report and the cost statement use it. `extractive` costs nothing |
| `embed.provider` | string | no, `hashing` | `hashing` (the local 64-dimension embedder), or `openaicompat`: `POST {embed.base}/embeddings` with `embed.model`, no key, returning `embed.dim` numbers a text (SB-1710). A different embedder than the one the index was built with rebuilds the index (sb-ingest(7)) |
| `embed.base`, `embed.model`, `embed.dim` | URL, string, int | with `openaicompat` | The embeddings endpoint (on a self-hosted cell `http://model-embed:8080/v1`, on the `data` network), the model's id, and its vector size (1 to 4096) |
| `connectors.KIND` | object | no | Overrides for one connector: `base` (the vendor API's address, for a sandbox or a recorded fake). For `docs: folder` it is required and names the directory, `file:///DIR`; the connector keeps its listings under `state/connectors/KIND/` (sb-connector(3), SB-1606) |
| `microsoft.base`, `microsoft.login` | URLs | no | (SB-5901 to SB-5907) Microsoft Graph's API and Entra's sign-in, for a fake in tests; default `https://graph.microsoft.com/v1.0` and `https://login.microsoftonline.com`. The app's credentials are at `vault://tenants/TENANT/msgraph`, `{ "tenant_id", "client_id", "client_secret" }` (ADR-0023) |
| `microsoft.drives`, `microsoft.organizers`, `microsoft.domain` | lists, string | with `docs: sharepoint` | (SB-5901, SB-5906) The drive ids the sharepoint connector reads (a site's library or a OneDrive), the Entra user ids whose Teams meetings' transcripts it reads, and the company's email domain an organization sharing link grants (sb-connector(3), `docs: sharepoint`) |
| `labels.microsoft` | map label id → level | no | (SB-5902) Purview sensitivity label ids to `public`, `internal`, `confidential` or `restricted`. An id it doesn't map counts as `restricted`, and the health report lists it by id and count |
| `teams.app_id`, `teams.openid`, `teams.login` | strings, URLs | no | (SB-5904) The Teams bot's Microsoft app id, its password at `vault://tenants/TENANT/teams` (`{ "app_id", "app_password" }`), and the Bot Framework's OpenID configuration and token addresses, for a fake in tests ("" for Microsoft's). It needs `microsoft` to find people by their Entra ids (sb-ask(3), The chat app (Teams)) |
| `chat.save_reaction` | string | no | (SB-5806) The reaction that offers to save a thread or ask about it, in a direct message to whoever added it (sb-ask(3), The chat app (Slack)); default `brain` |
| `chat.team_id` | string | when `chat: slack` | The tenant's Slack workspace id: the chat app (sb-ask(3)) answers only requests from it, and `POST /connectors/slack/events` takes its events |
| `mattermost.base`, `mattermost.team_id` | https URL, string | no | The Mattermost server and the tenant's team for the Mattermost slash command at `POST /chat/mattermost` (sb-ask(3), The chat app (Mattermost); SB-2005), with `{"command_token", "bot_token"}` at `vault://tenants/TENANT/mattermost`. The same bot sends notifications as direct messages (sb-ask(3), Notifications). Without them nothing answers there. The base is `https`, or `http` on `127.0.0.1` only for a server on this machine |
| `calendar.kind`, `calendar.base` | `google`, https URL | no | (SB-4202) A read-only calendar for meeting prep (sb-ask(3), Meeting prep; sb-connector(3), Calendars): each person's own events, read as them through the Google service account at `vault://tenants/TENANT/calendar` with domain-wide delegation for the one scope `calendar.events.readonly`. `base` defaults to `https://www.googleapis.com`. Without it there is no calendar, and prep takes a pasted agenda only |
| `email.server`, `email.from` | `HOST:PORT`, address | no | The company's mail server and the sender address for notifications by email, on a cell with neither `mattermost` nor a `chat` connector (sb-ask(3), Notifications; SB-2307). The cell always says `STARTTLS` and refuses a server that doesn't offer it, verifying the server's certificate for `HOST` against the system's roots and `/etc/sb/ca`. A server that wants a sign-in reads `{"username", "password"}` from `vault://tenants/TENANT/email`; with no such secret the cell sends without one. Port 587 is usual |
| `alerts.webhook` | https URL | no | Where alerts are posted as JSON (sb-observability(7)); every alert is also logged and counted in `sb_alerts_total` |
| `alerts.report_webhook` | https URL | no | Where the weekly report is posted (sb-observability(7), Weekly leadership report); without it, `alerts.webhook` (SB-1608) |
| `audit.stream` | URL | no | `syslog+tls://HOST:PORT`: forward every audit row to the company's SIEM as it is written, RFC 5424 over TLS, the row as the export has it; the SIEM's certificate is verified against the system's roots and `/etc/sb/ca`. A SIEM that is down never blocks a write: rows queue up to a day, a Sev-2 alert fires after an hour, and the stream resumes from the last row the SIEM took (sb-audit(7), Streaming to a SIEM; SB-2302) |
| `backups` | list | for the `backup` job | The nightly backup's copies, each `{dir, media, offsite}` (with `seal: "sb-recovery-pub:…"`, every object sealed to that recovery key before it is written there, as `NAME.sealed`; SB-1901) or `{worm: true}` for the `worm` store, or `{s3: "s3://BUCKET/PREFIX", offsite: true, seal: "sb-recovery-pub:…"}` for a company bucket with no object lock, its key pair (with `endpoint` and `region` for a store other than AWS) at `vault://tenants/TENANT/offsite`, sealed always, each object written once (SB-2303): at least three, on two kinds of media, one off-site (sb-audit(7)). Without them the `backup` job fails, and its missed run is a Sev-1 |
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
