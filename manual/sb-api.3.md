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

Session tokens come from the tenant's OIDC sign-in. The token's `tenant` claim must equal the cell's tenant, or the request fails with `ETENANT`. Sessions are re-validated against the directory every 60 seconds, so SCIM deactivation takes effect within 5 minutes (LAW-12). Service integrations (the chat-app bot) use workload identity, never static API keys.

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
