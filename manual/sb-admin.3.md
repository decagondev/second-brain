---
title: "sb-admin(3)"
man: "3"
summary: Admin API used by the console and by automation.
---

# sb-admin(3)

## NAME

sb-admin — manifest diffs, kill switches, audit and usage

## SYNOPSIS

```http
GET  /v1/admin/manifest
POST /v1/admin/manifest/diff
POST /v1/admin/kill
POST /v1/admin/unkill
GET  /v1/admin/kill
GET  /v1/admin/audit?since=…&kind=…
GET  /v1/admin/usage?week=2027-W10
GET  /v1/admin/refusals?since=…
```

## DESCRIPTION

The console is a client of this API and has no other powers. Every endpoint needs the `operator` role (Cedar policy) except `usage`, which `approver` may also read.

**`GET /v1/admin/manifest`** returns the applied manifest and its git commit.

**`POST /v1/admin/manifest/diff`** takes a JSON patch, e.g. `[{ "op": "add", "path": "/agents/-", "value": "draft-reply" }]`, validates it as `sb tenant check` would, and returns the YAML diff. Depending on tenant setup, the diff is either opened as a pull request against the tenant's manifest repo or signed and applied. Either way, the result is a manifest change (LAW-16).

**`POST /v1/admin/kill`** takes `{ "scope": "agent"|"connector"|"tenant", "name": "…", "reason": "…" }` and has the same semantics as [sb-kill(1)](sb-kill.1.md).

**`GET /v1/admin/audit`** pages through [sb-audit(5)](sb-audit.5.md) rows.

**`GET /v1/admin/usage`** returns weekly active users, seats, asks, cited-answer rate, refusal rate by connector, drafts, approve-to-reject ratio and model spend.

**`GET /v1/admin/refusals`** lists refusal counts by connector and reason (`deny`, `unknown`, `timeout`, `missing`), with no document titles, so operators can spot a misbehaving connector.

## ERRORS

`EPERM` without the role · `EMANIFEST` for an invalid patch · `EDRIFT` if the live config has drifted and a diff can't be based on it.

## LAWS

[LAW-10](sb-laws.7.md#law-10), [LAW-16](sb-laws.7.md#law-16).

## SEE ALSO

[sb-tenant(5)](sb-tenant.5.md), [sb-kill-switches(7)](sb-kill-switches.7.md), [sb-observability(7)](sb-observability.7.md)
