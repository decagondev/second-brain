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
POST /v1/admin/subjects/erase
```

## DESCRIPTION

The console is a client of this API and has no other powers. Every endpoint needs the `operator` role (Cedar policy) except `usage`, which `approver` may also read.

**`GET /v1/admin/manifest`** returns the applied manifest and its git commit.

**`POST /v1/admin/manifest/diff`** takes a JSON patch, e.g. `[{ "op": "add", "path": "/agents/-", "value": "draft-reply" }]`, validates it as `sb tenant check` would, and returns the YAML diff. Depending on tenant setup, the diff is either opened as a pull request against the tenant's manifest repo or signed and applied. Either way, the result is a manifest change (LAW-16).

**`POST /v1/admin/kill`** takes `{ "scope": "agent"|"connector"|"tenant", "name": "…", "reason": "…" }` and has the same semantics as [sb-kill(1)](sb-kill.1.md).

**`GET /v1/admin/audit`** pages through [sb-audit(5)](sb-audit.5.md) rows.

**`GET /v1/admin/usage?week=2027-W10`** returns the weekly report's numbers (sb-observability(7), Weekly leadership report): weekly active users, seats, asks, cited-answer rate, gaps by kind, drafts and their used-to-discarded ratio, actions by outcome and model spend.

**`GET /v1/admin/refusals`** lists refusal counts by connector and reason (`deny`, `unknown`, `timeout`, `missing`), with no document titles, so operators can spot a misbehaving connector.

**`POST /v1/admin/subjects/erase`** takes `{ "user": "usr_…", "reason": "…" }` and erases the user's derived data as sb-tenant(5), Retention and deletion, describes. The response is `{ "erased": { "traces": 3, "conversations": 1, "feedback": 1, "logs": 12 }, "audit": ["aud_…", "aud_…"] }`. A user id that isn't `usr_` followed by `[a-z0-9_]` is `EINVAL`, and a missing reason is `EINVAL`. Erasing a user with nothing stored succeeds with zero counts.

### Roles, exactly (SB-603)

| Role | Who | May |
|---|---|---|
| `operator` | members of the tenant's IdP group `sb-operators` | every admin endpoint |
| `approver` | members of any workflow's approval group in the manifest | `usage`, and approving actions (sb-approvals(7)) |
| everyone else | | none of these endpoints |

Roles are Cedar policies, not code: the base policy permits `admin` on the tenant's own `Cell` only for a principal in `Group::"sb-operators"`, with the tenant condition every permit carries (sb-policy(5)). A tenant policy can narrow it but not widen it. Groups come from the cell's own directory, so an operator of one tenant has no role in another (LAW-02).

### The manifest diff, exactly (SB-602)

`POST /v1/admin/manifest/diff` accepts JSON Patch operations (`add`, `remove`, `replace`) on these paths only: `/agents`, `/workflows`, `/connectors/<kind>`, `/approval/<workflow>`, `/exclude/<kind>`, `/retention_days`, `/budget_usd_month`, `/pilot_group`, `/channel`. `tenant`, `region` and `idp` can't be patched here. The patched manifest is checked exactly as `sb tenant check` checks a file. The response is `{ "manifest": "<the whole new manifest as YAML>", "diff": ["- agents: [ask]", "+ agents: [ask, draft-reply]"], "plan": "<the sb tenant apply --plan output>" }`. It **changes nothing**: every proposal writes a `kind: config` audit row with the operator and the patch's hash, and the change takes effect only when the new manifest is applied. Kill switches offer **make permanent**, which is this same endpoint with a patch that removes the agent or connector.

## ERRORS

`EPERM` without the role · `EMANIFEST` for an invalid patch · `EDRIFT` if the live config has drifted and a diff can't be based on it.

## LAWS

[LAW-10](sb-laws.7.md#law-10), [LAW-16](sb-laws.7.md#law-16).

## SEE ALSO

[sb-tenant(5)](sb-tenant.5.md), [sb-kill-switches(7)](sb-kill-switches.7.md), [sb-observability(7)](sb-observability.7.md)
