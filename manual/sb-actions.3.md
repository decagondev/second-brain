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

How a draft is made, exactly (SB-409):

1. **Validate.** `context` is `chat:<channel>/<ts>` or `tickets:<KEY-123 or id>`, and `instruction` is 1 to 2,000 characters. Anything else is `EINVAL`.
2. **Kill and policy.** The `draft-reply` agent must not be killed (`EKILLED`), and policy must allow `use_agent` on `Agent::"draft-reply"` with `context.target_project` set to the ticket's project key (`FIN` for `FIN-12`), so tenant policies can narrow drafting by project (`EPERM`).
3. **The context item, live.** The item is checked with a live `ACL` call like any chunk. Anything but Allow is `EPERM`, and nothing about the item is returned. Its text becomes the first data block, `C1`, citable like any source.
4. **Retrieve, compose.** Retrieval and the live check run as for `ask`, with the instruction as the query. The bundled `draft` prompt writes the reply, and the composer decides citations (sb-citations(7)).
5. **Never sent.** The draft agent has no tools. The response has the same shape as an answer (`trace`, `release`, `answer.claims`, `refused`, `notice`) plus `context`. The UI offers **Copy**, and the person sends it in the target app.

`POST /v1/drafts/{trace}/outcome` takes `{ "outcome": "used" | "discarded" }` from the person who drafted. It is counted in `sb_drafts_total{outcome}`. That is the approve and reject rate leadership reviews at day 45 (sb-onboarding(7)).

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

### Draft a Jira issue (SB-6207)

`POST /v1/actions/draft { "trace", "kind": "jira-issue" }` drafts a new Jira issue from the caller's own answer, its cites checked again as them (a cite they can no longer open is left out, with its claim): `{ "kind": "jira-issue", "summary", "text", "cites", "link", "warning"? }`. `link` opens Jira's own create form (`/secure/CreateIssueDetails!init.jspa`) in the first project of sb-cell(5) `atlassian.projects`, filled in with the summary and the text, so the person files it under their own sign-in; the cell never files an issue and keeps no token of theirs. `warning` says the draft cites a confidential or restricted document the project's readers may not open (LAW-32). Without Jira it is `EUNAVAIL`. The web UI's **Draft a Jira issue** shows the text, the warning and **File it in Jira**.

### Draft the next action (SB-5304)

An answer can become the next action, drafted for the person to edit; nothing goes out until they press **Send**.

- `POST /v1/actions/draft` `{ "trace": "tr_…", "kind": "ticket"|"message" }`, on the caller's own answer, returns `{ "kind", "text", "cites" }`: each cited claim as a line, then the cites, every one checked again as the caller now; a claim whose cite they can no longer open is left out (LAW-26). Nothing is sent or kept.
- A **ticket** comment is sent as any action is: `POST /v1/actions` with `workflow: ticket-update`, the ticket as `target` and `{ "comment": TEXT }`, so it waits for its approver and is audited like every action (Actions, above; LAW-04).
- A **message** is posted by the cell's chat bot (Mattermost, else Slack) in one of the space's `channels` (sb-space(5)): `POST /v1/spaces/NAME/messages` `{ "channel": "#ops", "text": "…" }` (1 to 4,000 characters), for a member of the space, answers `{ "channel", "id" }`. It starts "Sent by EMAIL via Second Brain:", is masked like everything the cell sends (sb-ingest(7), Nothing sensitive goes out), and is audited `message_sent` with its hash before it is posted. A channel that isn't the space's is `EINVAL`; a cell without a chat bot is `EUNAVAIL`. The bot must be a member of the channel.

The web app's **Draft a ticket comment** and **Draft a message** under each answer do this, with the text to edit and **Send**.

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
