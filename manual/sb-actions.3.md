---
title: "sb-actions(3)"
man: "3"
summary: Drafts, proposed actions and approvals.
---

# sb-actions(3)

## NAME

sb-actions — `POST /v1/drafts`, `POST /v1/actions`, `GET /v1/actions/{id}`, `POST /v1/actions/{id}/approve`, `POST /v1/actions/{id}/reject`

## SYNOPSIS

```http
POST /v1/actions
Idempotency-Key: 01JD3QE2…
{ "workflow": "ticket-update", "target": "tickets:OPS-1182", "payload": { "comment": "…" } }
```

## DESCRIPTION

### Drafts (`draft-reply`)

`POST /v1/drafts` with `{ "context": "chat:C024BE91L/1709550000.1234" | "tickets:OPS-1182", "instruction": "…" }` returns a draft with citations, exactly like an answer. The platform **never sends** a draft. The user copies it, or opens it in the target app, and sends it there. That's why drafts need no approval.

### Actions (gated workflows)

`POST /v1/actions` proposes a consequential action for the tenant's enabled workflow. The response is the action in state `PendingApproval`:

```json
{
  "id": "tc_01JD3QC7…",
  "state": "PendingApproval",
  "workflow": "ticket-update",
  "target": "tickets:OPS-1182",
  "payload": { "comment": "Root cause: …" },
  "payload_sha256": "5d1f0c…",
  "requester": "usr_01JD3Q8Z…",
  "approvers": "group:it-leads",
  "expires_at": "2027-03-03T10:41:07Z"
}
```

Approvers fetch it with `GET /v1/actions/{id}`, which adds a `diff` against the target's current state where the connector supports it.

`POST /v1/actions/{id}/approve` **must** echo the hash the approver was shown:

```json
{ "payload_sha256": "5d1f0c…", "note": "ok" }
```

If the hash doesn't match the current payload, approval fails with `ENOAPPROVAL`. This makes it impossible to approve something other than what was displayed (LAW-15). The approver must not be the requester.

On approval, the executor runs the action. Poll `GET /v1/actions/{id}` or subscribe to the webhook. The final state is `Ran` (with `receipt`) or `DidNotRun` (with `reason`).

`POST /v1/actions/{id}/reject` takes `{ "note": "…" }`.

### States

`Drafted`, `PendingApproval`, `Approved`, `Rejected`, `Expired`, `Executing`, `Ran`, `DidNotRun`. See [sb-approvals(7)](sb-approvals.7.md).

## ERRORS

| Code | When |
|---|---|
| `EPERM` | Workflow not enabled, user not allowed to request, or approver not in the approver group |
| `ENOAPPROVAL` | Hash mismatch, expired, self-approval, or execute without approval |
| `EKILLED` | Agent, connector or tenant killed |
| `ENORECEIPT` | Reported on the action as `DidNotRun` when the target returned no receipt |

## LAWS

[LAW-04](sb-laws.7.md#law-04), [LAW-05](sb-laws.7.md#law-05), [LAW-10](sb-laws.7.md#law-10), [LAW-15](sb-laws.7.md#law-15).

## EXAMPLES

```sh
ID=$(curl -sS -X POST "$SB/v1/actions" -H "Authorization: Bearer $TOKEN" \
      -H "Idempotency-Key: $(ulid)" -d @proposal.json | jq -r .id)
HASH=$(curl -sS "$SB/v1/actions/$ID" -H "Authorization: Bearer $APPROVER" | jq -r .payload_sha256)
curl -sS -X POST "$SB/v1/actions/$ID/approve" -H "Authorization: Bearer $APPROVER" \
      -d "{\"payload_sha256\":\"$HASH\"}"
```

## SEE ALSO

[sb-approvals(7)](sb-approvals.7.md), [sb-agents(7)](sb-agents.7.md), [sb-connector(3)](sb-connector.3.md)
