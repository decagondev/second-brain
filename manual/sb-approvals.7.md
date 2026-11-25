---
title: "sb-approvals(7)"
man: "7"
summary: The action state machine, payload binding and separation of duties.
---

# sb-approvals(7)

## NAME

sb-approvals — how a proposed action becomes a real change, or doesn't

## SYNOPSIS

```text
Drafted → PendingApproval → Approved → Executing → Ran | DidNotRun
                         ↘ Rejected | Expired | (edit) → Drafted
```

## DESCRIPTION

Only **gated workflows** produce actions. Drafts from `draft-reply` are not actions, because a human sends them.

### States

| State | Meaning | Next |
|---|---|---|
| `Drafted` | Agent has a proposed payload | `PendingApproval` |
| `PendingApproval` | Audit row written; payload hash fixed; approvers notified | `Approved`, `Rejected`, `Expired`, or `Drafted` on edit |
| `Approved` | An approver confirmed **this hash** | `Executing`, `Expired` |
| `Rejected` | An approver declined | end |
| `Expired` | TTL passed (default 24 h) | end |
| `Executing` | Kill re-checked; audit row written; `Act` called | `Ran`, `DidNotRun` |
| `Ran` | Target returned a receipt | end |
| `DidNotRun` | Error, timeout or no receipt | end |

These names are identical in code, in the API and in [`proofs/tla/Approvals.tla`](../proofs/tla/Approvals.tla).

### Rules

1. **Audit first.** Entering `PendingApproval` and entering `Executing` each write an audit row before anything else happens (LAW-04).
2. **One payload.** The approval stores the SHA-256 of the canonical payload. Approving requires echoing that hash. Executing requires the current payload to hash to it. Any edit returns to `Drafted` (LAW-15).
3. **Not your own, nor your report's.** The approver must not be the requester, nor anyone above the requester in the directory's reporting line (LAW-15). A workflow with a duty above its threshold needs two approvers from different groups (Segregation of duties, below).
4. **Kill wins.** The kill switch is checked when approved and again just before `Act` (LAW-10).
5. **Receipt or it didn't happen.** Only a target receipt makes `Ran` (LAW-05).
6. **At most once.** The approval id is the idempotency key for `Act`.

An agent's draft in a steward queue is not an action and has no approval: it publishes only when a steward replies with it and publishes it as a note (sb-spaces(7)). **Ask for changes** (SB-4602; sb-agents(7)) sends a draft back for a redraft, at most 3 rounds, before that; a redraft is still a draft, and nothing it says is published, sent or run until the steward publishes it (LAW-16).

### The `ticket-update` workflow

The first gated workflow (SB-410). Enabling it takes `workflows: [ticket-update]` and `approval: { ticket-update: group:<name> }` in the manifest, the `tickets` connector, and a passing eval record (sb-tenant(1)).

| Part | Value |
|---|---|
| Tool | `ticket-update.comment`, connector `tickets`, consequential |
| Target | `tickets:<KEY-123>` |
| Payload | `{"comment": "…"}`: one string, 1 to 10,000 characters, no other fields |
| Approvers | The manifest's approval group |
| Receipt | The target's comment id, for example `linear:comment/<id>` |

### The payload hash, exactly

`payload_sha256` is the hex SHA-256 of the payload's RFC 8785 canonical JSON: object keys sorted, no whitespace, strings in their shortest JSON escaping. Numbers must be integers between -2^53 and 2^53. Any other number is refused when the action is proposed, so every implementation hashes the same bytes. The hash covers the payload **and** the target (`{"payload":…,"target":"tickets:OPS-1182","workflow":"ticket-update"}`), so approving a comment for one ticket can't post it on another.

### Approving, exactly

`approve` succeeds only if all of these hold, checked in this order, and each failure is `ENOAPPROVAL` with a reason: the action is `PendingApproval`, it hasn't expired (`expires_at`, 24 hours after proposal by default), the echoed hash equals the current payload's hash, the approver isn't the requester or above them in the reporting line, isn't the action's first approver, and policy allows `approve` (which checks the approver group: the workflow's for the first approval, the duty's `second` group for the second). The approval gets an id (`apr_…`) recorded with the hash and the approver, and then the executor runs. `reject` needs the same state and policy, takes a note, and ends the action. A note may be empty, but the click is always explicit.

### Segregation of duties, exactly (SB-1204)

"Approver isn't requester" isn't enough for finance controls. A workflow can carry a **duty** in the manifest (sb-tenant(5), `duties`):

```yaml
duties: { ticket-update: { second: group:finance-leads, above: 100000 } }
```

- **Two approvers above the threshold.** An action whose payload's `amount` (a whole number in minor units, such as cents) is above `above` needs two approvals: the first from the workflow's `approval` group and the second from the duty's `second` group, by two different people. The size of the amount decides, whatever its sign, so a refund of a million needs both like a payment of a million. An action whose payload has no `amount`, or one that isn't a whole number, needs both: an unknown amount never needs fewer approvals. `above: 0` means every action needs both. An edit decides again from the new payload, so raising the amount above the threshold needs both approvals again (SB-1401).
- **Not the requester's manager.** Neither approver may be the requester or anyone in the requester's reporting line: their manager, their manager's manager, and so on, from the directory's `manager` attribute (SCIM enterprise extension, sb-identity(7)), read when the approval is made. A reporting line the directory can't read, or one longer than 20, refuses the approval (`ENOAPPROVAL`), and so does a cell that can't read reporting lines at all when the action is under a duty. A rejection takes the same checks as an approval: the same group for the turn, not the requester, not their reporting line (SB-1401).
- **The group decides, whatever else permits.** A platform rule refuses an approval by anyone outside the group named for it (the workflow's group first, the duty's `second` group second), even where a tenant's own policy permits approving more widely. A law test found exactly that gap: a tenant permit for IT leads let one of them give the finance approval.
- **One hash, both approvals.** Each approver echoes the same payload hash. The first approval is recorded (`kind: approval`, `state: first_approval`) and the action stays `PendingApproval`; the second makes it `Approved` and the executor runs. Any edit returns the action to `Drafted` and voids both. Expiry counts from the proposal, not from the first approval, and an edit never moves it (SB-1401).
- **In the model.** `proofs/tla/Approvals.tla` checks `NoSelfApproval`, `NoManagerApproval` and `TwoApprovers` for every interleaving, beside the earlier invariants.

The response of `approve` says which approval it was: `"approvals": 1` and `"needs": 2` after a first approval, `"approvals": 2` when it runs.

### Timeouts with unknown effect

If `Act` times out, the action is `DidNotRun` and the user is told the result couldn't be confirmed. A reconciler asks the target by idempotency key. If it finds the change, it appends a `Ran` audit row with the receipt and notifies the user. It never re-executes.

Exactly (SB-408):

- **Idempotency store.** Every receipt is stored under its approval id. Before `Act`, the executor looks the approval id up. If there's a receipt, the action is `Ran` with that receipt and `Act` isn't called again. Losing the action's state can therefore never run it twice.
- **Unconfirmed.** An `Act` that returns an error or times out leaves the action `DidNotRun` with `unconfirmed: true`. A target that answered with no receipt is `DidNotRun` and confirmed: nothing to reconcile.
- **Reconciling.** Every 5 minutes for 24 hours, the reconciler asks the connector (`Find` by idempotency key) about each unconfirmed action. If the target has the change, the reconciler writes a `Ran` audit row with the receipt, stores the receipt, records it on the action (`receipt`, `reason: reconciled`, `unconfirmed: false`) and notifies. The action's state stays `DidNotRun`, because `DidNotRun` is terminal in the model, so the API reports the receipt alongside it. After 24 hours with nothing found, it records `reason: not found after 24h` and stops. A connector without `Find` leaves the action unconfirmed for a person to check.

## LAWS

[LAW-04](sb-laws.7.md#law-04), [LAW-05](sb-laws.7.md#law-05), [LAW-10](sb-laws.7.md#law-10), [LAW-15](sb-laws.7.md#law-15).

## EXAMPLES

```text
10:12  tc_01JD3QC7  Drafted          ticket-update OPS-1182 by usr_dana
10:12  tc_01JD3QC7  PendingApproval  hash 5d1f0c…  approvers group:it-leads
10:31  apr_01JD3QD9 Approved         by usr_erin, hash 5d1f0c… ✓
10:31  tc_01JD3QC7  Executing        kill check ok
10:31  tc_01JD3QC7  Ran              receipt jira:comment/883104
```

## SEE ALSO

[sb-actions(3)](sb-actions.3.md), [sb-agents(7)](sb-agents.7.md), [09 Approvals and actions](../docs/architecture/09-approvals-and-actions.md)
