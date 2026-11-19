---
title: "sb-laws(7)"
man: "7"
summary: The eighteen laws the platform must never break, and how each is checked.
---

# sb-laws(7)

## NAME

sb-laws — the platform's invariants, how each is checked, and where

## SYNOPSIS

```text
LAW-01 cite only what the asker can open     LAW-09 data is not instructions
LAW-02 no tenant crossing                    LAW-10 kill wins
LAW-03 unknown means refuse                  LAW-11 audit chain unbroken
LAW-04 audit before act                      LAW-12 deprovision revokes fast
LAW-05 ran means receipt                     LAW-13 the index is derived
LAW-06 only signed, gated builds run         LAW-14 every claim cited or labelled
LAW-07 no secret in prompt, log or audit     LAW-15 approval binds one payload
LAW-08 at most one behind stable             LAW-16 the manifest is the truth
LAW-17 hand-off leaves no unlisted access    LAW-18 the root key always has a custodian
```

## DESCRIPTION

A **law** is a statement the platform guarantees for every input, every tenant and every release. Laws are written here first, checked by tests, models, evals and production monitors, and tracked in the [assurance ledger](../living/ASSURANCE_LEDGER.md). A law violation in production is **Sev-1**, always.

Each law lists its **checks**:

| Check | Meaning |
|---|---|
| **Model** | A TLA+ or Alloy model in `proofs/` checks it over all small cases |
| **Property** | A property-based test in `laws/` checks it over generated cases |
| **Fault** | A fault-injection test checks it under connector failures |
| **Eval** | An eval suite measures it where the model's behaviour matters |
| **Verify** | `sb tenant verify` checks it on a tenant's real data |
| **Watch** | A production monitor alerts on it |

### LAW-01: Cite only what the asker can open {#law-01}

No answer cites, quotes or uses a chunk from a document that the asker can't open **at answer time**, according to the source's live ACL.

*Checks:* Model (`Permissions.als: CitedImpliesLiveAllow`) · Property, which asserts the source **was asked**, not just that the prefilter matched · Fault · Eval (`permission`) · Verify · Watch (sampled re-checks of citations).

### LAW-02: No tenant crossing {#law-02}

No retrieval, query, cache or log read returns data from another tenant.

*Checks:* Model (`NoCrossTenant`) · Property with two seeded tenants on every CI run · adversarial queries · Verify · Watch (`ETENANT` counter).

### LAW-03: Unknown means refuse {#law-03}

An `Unknown` permission result (timeout, error, missing source, unmapped model) is treated exactly like `Deny`.

*Checks:* Model (`UnknownNeverUsed`) · Fault: full matrix through the mock connector · Verify (injects a timeout) · lint: exhaustive `Decision` switches.

### LAW-04: Audit before act {#law-04}

Every tool call has a committed audit row **before** it executes. A consequential call also has a valid approval id in that row.

*Checks:* Model (`Approvals.tla: AuditBeforeExec`, `ApprovalBeforeExec`) · Property: audit replay reconstructs every call · Watch: audit-insert failures.

### LAW-05: Ran means receipt {#law-05}

An action is reported as `Ran` only if the target returned a receipt. Otherwise it is `DidNotRun`.

*Checks:* Model (`RanImpliesReceipt`) · Fault: a mock with no receipt must yield `DidNotRun` · contract kit `act/receipt-or-error`.

### LAW-06: Only signed, gated builds run {#law-06}

The build running in every cell is a signed release whose bundle passed the eval gate.

*Checks:* Model (`Rollout.tla: OnlyGatedRuns`) · deploy-time verification · Watch: hourly digest attestation per cell.

### LAW-07: No secret in prompt, log or audit {#law-07}

No secret (credential, token, key) appears in any prompt, log line or audit row.

*Checks:* canary secrets in vault and fixtures · scanner over prompts, logs and audit in CI · Watch: the same scanner in production.

### LAW-08: At most one behind stable {#law-08}

No stable-channel tenant on the release train (custody `factory` or `tenant`) runs a release more than one behind the current stable. Detached cells are off the train ([sb-factory(7)](sb-factory.7.md)).

*Checks:* Model (`Rollout.tla: OneBehindMax`). Removing the promote guard produces a counterexample · release-controller check on every promotion · Watch: hourly lag.

### LAW-09: Data is not instructions {#law-09}

Ingested content can't cause a tool call or exfiltrate data. Only the user's own request can lead to a tool call.

*Checks:* Eval (`injection`, zero tool calls and zero canary leaks) · Property: tool-argument provenance never from flagged spans.

### LAW-10: Kill wins {#law-10}

Within 10 seconds of a kill switch being set, no new tool call or run starts in its scope.

*Checks:* Model (`KillWins`: once killed, no `Executing` transition) · timed test · drill every release.

### LAW-11: Audit chain unbroken {#law-11}

The audit log is append-only. Every row's `prev` equals the previous row's `hash`, and every hourly anchor verifies.

*Checks:* tamper test · `sb audit verify` in CI, after restore drills and hourly in production.

### LAW-12: Deprovision revokes fast {#law-12}

Within 5 minutes of a SCIM deactivate, no request from that user succeeds.

*Checks:* timed property test · Watch: requests by deactivated users.

### LAW-13: The index is derived {#law-13}

Dropping and rebuilding a tenant's index from its sources yields the same set of `(cite, ACL)` pairs.

*Checks:* scheduled rebuild drill with set equality · Property on the ingest pipeline (idempotent per `(cite, span)`).

### LAW-14: Every claim cited or labelled {#law-14}

Every claim in an answer or draft either cites an allowed chunk or carries the `uncited` label, set by code.

*Checks:* Property over the composer · Eval (`citation`) · Verify.

### LAW-15: Approval binds one payload {#law-15}

An approval authorises exactly one payload, identified by its hash, approved by someone other than the requester. Any edit voids it.

*Checks:* Model (`ExecutedMatchesApproved`, `NoSelfApproval`) · Property: hash mismatch always refuses.

### LAW-16: The manifest is the truth {#law-16}

A cell's live configuration equals its applied manifest. Every console change is a manifest diff.

*Checks:* Watch: hourly drift detector · Property: console actions produce manifest diffs.

### LAW-17: Hand-off leaves no unlisted access {#law-17}

After a hand-off to `tenant` custody completes, the factory holds no access path to the cell (IAM role, deploy credential, break-glass role, network route, key wrap) except the grants listed in the hand-off record.

*Checks:* Model (`Handoff.tla: NoUnlistedAccess`) · Verify (`sb tenant verify --law LAW-17`, which enumerates access paths from IaC state and IAM) · Watch: monthly re-enumeration. Planned for W7; v1 has no hand-offs.

### LAW-18: The root key always has a custodian {#law-18}

At every moment, at least one custodian can unwrap a cell's root key. A hand-off re-wraps under the new custodian **before** destroying the old wrap.

*Checks:* Model (`Handoff.tla: KeyNeverOrphaned`) · step-ordering test in the hand-off controller · Watch: key-wrap inventory per cell.

## SEE ALSO

[sb-factory(7)](sb-factory.7.md), [Assurance ledger](../living/ASSURANCE_LEDGER.md), [04 Testing and assurance](../docs/04-testing-and-assurance.md), [proofs/](../proofs/README.md)
