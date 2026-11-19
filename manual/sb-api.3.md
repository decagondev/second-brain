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

Each tenant's cell serves its own API at its own hostname. There is no shared, multi-tenant API endpoint for content.

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

### Idempotency

POSTs that create drafts or actions accept `Idempotency-Key`. Repeating a request with the same key within 24 hours returns the original response and creates nothing new.

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
