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

`POST /v1/actions` takes `{ "workflow": "ticket-update", "target": "tickets:OPS-1182", "payload": { … } }`. The tool is the one the release's registry lists for that workflow; a workflow that isn't enabled is `EPERM`.

`GET /v1/actions/{id}` may be read by the requester, by members of the approver group, and by auditors. Anyone else gets `EPERM`, the same as an unknown id. Besides the fields above, it returns:

| Field | Meaning |
|---|---|
| `policy_reason` | Why policy allowed the proposal, e.g. `propose: permitted by base.cedar#10` |
| `diff` | `[{ "field", "before", "after" }]`, one per top-level payload field that differs from the target's current state, or `null` when the connector can't read the current state. Values are shown exactly as stored |
| `approval`, `approved_by` | Set once approved |
| `receipt` / `reason` | Set once `Ran` / `DidNotRun` |

### The approver view

The web UI shows an action at `/?action=tc_…` (SB-407). It shows the target system and object, the requester, the policy reason, the expiry, the exact payload, and the diff, field by field. The page shows the hash it was given and **echoes that same hash** on approve, so a payload that changed after the page loaded is refused (`ENOAPPROVAL`) rather than approved unseen. Approve and Reject are two separate buttons. Each needs an explicit click, and each offers an optional note. The page never approves on load, on Enter, or for a list of actions at once.

Approvers are told about new actions through the cell's notification hook (chat or email adapter). The notification carries the action id and workflow only, never the payload.

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
