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
3. **Not your own.** The approver must not be the requester (LAW-15).
4. **Kill wins.** The kill switch is checked when approved and again just before `Act` (LAW-10).
5. **Receipt or it didn't happen.** Only a target receipt makes `Ran` (LAW-05).
6. **At most once.** The approval id is the idempotency key for `Act`.

### Timeouts with unknown effect

If `Act` times out, the action is `DidNotRun` and the user is told the result couldn't be confirmed. A reconciler asks the target by idempotency key. If it finds the change, it appends a `Ran` audit row with the receipt and notifies the user. It never re-executes.

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
