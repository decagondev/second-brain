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

The manifest (sb-tenant(5)) says what a tenant's brain is; this file says how one cell runs it: where its state lives, where its secrets are, what it listens on. The cell's services (`sb-api`, and from SB-1503 `sb-ingest` and `sb-jobs`) each read it with `--config FILE` and build every part of the cell from it in one place, `internal/cell/wire`, so no part is built twice or differently in two services.

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
| `connectors.KIND` | object | no | Overrides for one connector: `base` (the vendor API's address, for a sandbox or a recorded fake) |
| `chat.team_id` | string | when `chat: slack` | The tenant's Slack workspace id: the chat app (sb-ask(3)) answers only requests from it, and `POST /connectors/slack/events` takes its events |
| `durable` | bool | no, `false` | `true` refuses to start while any store is in-memory only |

### Stores

| Store | Adapter |
|---|---|
| audit log, kill switches, key wraps, sync cursors, logs, backups | files under `state` |
| directory and sessions, traces and conversations, actions and receipts, spaces registry, index, chat events inbox, proposals, access reviews, legal holds | in memory today; durable adapters are SB-1501's follow-up, and `durable: true` refuses to start until they exist |

### Connectors

Each kind in the manifest's `connectors` is served by the vendor's adapter: `docs: gdrive`, `chat: slack`, `tickets: linear`, `crm: hubspot`, `code: github`, `hris: bamboohr`, `accounting: quickbooks`. Credentials are read from the vault at each call (sb-connector(3)). Any other vendor stops the cell from starting.

## EXIT STATUS

`sb-api` exits 0 after a clean shutdown (SIGTERM or SIGINT: it stops taking requests and finishes those in flight), and 1 with the reason if it can't start.

## LAWS

[LAW-02](sb-laws.7.md#law-02): one file, one tenant, one cell. [LAW-03](sb-laws.7.md#law-03): nothing that can't be built is skipped. [LAW-16](sb-laws.7.md#law-16): the cell runs its manifest, nothing else.

## SEE ALSO

[sb-runtime(7)](sb-runtime.7.md), [sb-tenant(5)](sb-tenant.5.md), [sb-api(3)](sb-api.3.md), [sb-connector(3)](sb-connector.3.md)
