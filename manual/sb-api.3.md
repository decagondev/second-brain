---
title: "sb-api(3)"
man: "3"
summary: API conventions shared by every endpoint.
---

# sb-api(3)

## NAME

sb-api — authentication, versioning, errors, idempotency and pagination for the cell API

## SYNOPSIS

```text
https://<tenant>.<platform-domain>/v1/<resource>
Authorization: Bearer <session token>
Content-Type: application/json
Idempotency-Key: <client-chosen ULID>        (on POSTs that create actions)
```

## DESCRIPTION

Each tenant's cell serves its own API at its own hostname. There is no shared, multi-tenant API endpoint for content. The same host serves the web UI (SB-307) on `GET /` and every path outside `/v1`, so the browser app can route; an API path is never answered by the UI. `GET /healthz` and `GET /readyz` answer the platform's health checks with no content (sb-runtime(7)); `GET /login` and `GET /callback` are the sign-in (sb-identity(7)). Three routes answer the control plane only, by its identity (a bearer token equal to `vault://tenants/TENANT/control-plane`): the verifier's, `POST /v1/cell/manifest` (apply against a running cell, sb-tenant(1)) and `GET /v1/cell/live` (the running configuration in the plan's terms, for the drift check, sb-tenant(5)), and, for sb-audit(1) and detach (sb-factory(1)), `GET /v1/cell/audit?from=SEQ` (up to 1,000 rows and the next `seq`), `GET /v1/cell/audit/export?since=TIME&until=TIME` (the signed export) and `GET /v1/cell/package` (the detach package's files). Without the header they answer `EAUTH`, with another token `EPERM`. `sb-api` serves all of it from one configuration (sb-cell(5)).

### Authentication

Session tokens come from the tenant's OIDC sign-in. The token's `tenant` claim must equal the cell's tenant, or the request fails with `ETENANT`. Sessions are re-validated against the directory every 60 seconds, so SCIM deactivation takes effect within 5 minutes (LAW-12). Service integrations (the chat-app bot) use workload identity, never static API keys. A person's script uses a read-only API token (below), which acts as that person and only asks and searches.

**Read-only API tokens** (SB-2306). A script asks and searches as the person who made its token: `Authorization: Bearer sbt_…` on `POST /v1/ask`, `GET /v1/search` and, with `mcp: true`, `POST /v1/mcp` (MCP, below), and nothing else; any other path with a token is `EPERM`, so a token can't approve, act, administer, manage tokens or reach another cell. The person makes one in the web UI (**API tokens**, under the conversation) or with:

- **`POST /v1/tokens`** `{ "name": "nightly report", "days": 30 }` (1 to 90; default 30) answers `{ "id": "tok_…", "token": "sbt_…", "name", "created", "expires" }`. The token is shown **once**: the cell keeps only its SHA-256, as the key it looks the token up by, never the token.
- **`GET /v1/tokens`** lists the caller's tokens, `{ "items": [{ "id", "name", "created", "expires" }] }`, never a token. The web UI shows each one's days left, and seven days before a token expires the notify job tells its owner once, by the token's name (`tokens`, sb-ask(3), Notifications; SB-2606).
- **`DELETE /v1/tokens/{id}`** revokes one of the caller's tokens at once.

These three take a browser session only. A token is a session of its own kind: every request reads it from the store, checks the directory every 60 seconds and the person's groups and attributes at every use, exactly as a browser session does, and ends at its expiry, which is never extended. Signing out everywhere (below) and a SCIM deactivate end every token with the sessions (LAW-12). Making, using and revoking a token each write a `kind: access` audit row (subject `token:tok_…`, states `token_created`, `token_used` with the method and path, `token_revoked`); a use whose row can't be written is refused with `EUNAVAIL`. A use is still an ask or a search like any other: the same permission checks (LAW-01), the same `asks_per_minute` (sb-tenant(5)) and the same traces.

**MCP** (SB-3801). With sb-cell(5) `mcp: true`, `POST /v1/mcp` speaks the Model Context Protocol (JSON-RPC 2.0 over streamable HTTP, one JSON response per request, no event stream), so an AI tool the person uses can search and ask the brain **as them**. It takes the same `Authorization: Bearer sbt_…` token (or a browser session), and `?space=NAME` picks the space as on every other path. The methods are `initialize`, `ping`, `tools/list` and `tools/call`; a notification gets `202` and no body; a batch, or any other method, is a JSON-RPC error. There are three tools, each marked read-only: `search` `{ "query" }` (the results of `GET /v1/search`), `ask` `{ "question" }` (the answer of `POST /v1/ask`, with its cites and trace) and `passage` `{ "trace", "cite" }` (the cited text, as `GET /v1/traces/ID/passage` gives it). Each runs the same code as its path, so every result is checked live as the token's person, an ask counts toward `asks_per_minute`, and a refusal is the tool's error text (`isError: true`); none writes anything but the trace any ask or search writes. The token's use is audited as on the other paths (`token_used`, `POST /v1/mcp`), and a revoked or expired token, or a deactivated person, is refused at once (LAW-12). Without `mcp: true` the path isn't served.

**`POST /v1/sessions/revoke`** (SB-1906) ends every session of the caller, the one making the request included, and clears the session cookie: `{ "revoked": 2, "audit": ["aud_…", "aud_…"] }`. It takes no body. The next request on any of those sessions is `EAUTH` (sb-identity(7), Signing out everywhere). If the audit row can't be written first, nothing is revoked and the answer is `EUNAVAIL`.

### Versioning

The version is in the path (`/v1`). Within a version, changes are **additive only**: new fields and new endpoints. Clients must ignore fields they don't know. Removing or renaming anything is `/v2`.

### Errors

Every error has the same shape, and the `code` comes from [sb-errors(7)](sb-errors.7.md):

```json
{
  "error": {
    "code": "EPERM",
    "message": "You don't have access to the sources needed for this.",
    "trace": "tr_01JD3QB4…",
    "remedy": "Ask the document owner for access."
  }
}
```

Messages are safe to show to the user. They never include the titles or content of refused documents.

### Spaces

Every request is in one space or company-wide (sb-spaces(7)). A request names its space with the query parameter `space`, for example `POST /v1/ask?space=finance`. Without it, or with `space=company`, the request is company-wide. A space the cell doesn't hold, or one the asker isn't a member of, is `ESPACE` (403), never a company-wide request instead. `GET /v1/search` takes `space` the same way, and a saved question or share keeps the space it was made in. A space name is never read from a body or a header, so one request can't name two spaces.

`branch=true` with a department's `space` (SB-3903) adds the teams under it that the asker is a member of, each checked as its own space (sb-spaces(7), The tree). `GET /v1/spaces` lists the asker's spaces, `{ "spaces": [{ "name", "label", "parent"?, "teams"? }] }`: `parent` only when the asker is a member of the parent too, and `teams: true` on a space with at least one team the asker is in.

Under a space (SB-4101 to SB-4104): `GET /v1/spaces/NAME/goals`, `GET /v1/spaces/NAME/updates`, `GET|POST /v1/spaces/NAME/requests`, and `POST|GET /v1/spaces/NAME/policies`, `POST /v1/spaces/NAME/policies/read` and `GET /v1/spaces/NAME/policies/status` (sb-ask(3), Goals, updates and policies). Each is `EPERM` (403) for someone who isn't a member of the space, and `POST …/policies` and `GET …/policies/status` for someone who isn't its steward.

The legwork agents (SB-4201 to SB-4203): `POST|GET /v1/research`, `GET /v1/research/{id}`, `POST /v1/research/{id}/run`, `POST /v1/prepare`, `GET /v1/me/events`, `GET /v1/me/actions`, `POST /v1/me/actions/{id}/done`, and `POST|GET /v1/spaces/NAME/actions` (sb-ask(3), Research, meeting prep and action items), and `POST /v1/spaces/NAME/review/fix` (SB-4206; sb-ask(3), Follow-ups). Each is for the signed-in person only, and a stored result is checked again as them when it is read (LAW-26).

Work that comes to you (SB-4301 to SB-4303): `GET /v1/me/inbox`, `PUT /v1/saved/{id}/routine`, `GET|POST|DELETE /v1/me/memory` and `DELETE /v1/me/memory/{id}` (sb-ask(3), Work that comes to you). Each is for the signed-in person only; the inbox lists and never acts, and each item is checked as them when it is listed (LAW-26). A memory is never sent with anyone else's ask (LAW-25).

### Idempotency

POSTs that create drafts or actions accept `Idempotency-Key`. Repeating a request with the same key within 24 hours returns the original response and creates nothing new.

### Links for people

A share link is `/q/sh_…` on the cell's own hostname (sb-ask(3), Saved and shared questions). The web UI serves it, never `/v1`, and it holds no answer: the UI reads the share with `GET /v1/shares/{id}`, and the recipient asks it under their own access. A share id is 128 random bits, so one link can't be guessed from another.

### Pagination

List endpoints return `{ "items": [...], "next": "<cursor>" }`. Pass `?cursor=` to continue. Cursors are opaque and expire after 1 hour.

### Rate limits and budgets

`429` with `EBUDGET` when the tenant's `budget_usd_month` or the per-user rate limit is reached. `Retry-After` is set for rate limits, but not for budget exhaustion.

### Tracing

Every response carries `X-SB-Trace: tr_…` and `X-SB-Release: rel_…`.

## ERRORS

See [sb-errors(7)](sb-errors.7.md).

## LAWS

[LAW-02](sb-laws.7.md#law-02), [LAW-12](sb-laws.7.md#law-12).

## SEE ALSO

[sb-ask(3)](sb-ask.3.md), [sb-actions(3)](sb-actions.3.md), [sb-admin(3)](sb-admin.3.md), [sb-errors(7)](sb-errors.7.md)
