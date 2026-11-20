---
title: "sb-laws(7)"
man: "7"
summary: The twenty-three laws the platform must never break, and how each is checked.
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
LAW-19 spaces only narrow                    LAW-20 restricted stays home
LAW-21 moves revoke fast                     LAW-22 barriers hold
LAW-23 grants are consented and revocable
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

*Checks:* Model (`NoCrossTenant`) · Property with two seeded tenants on every CI run · adversarial queries, including hostile `Host` headers (SB-612) · Verify · Watch (`ETENANT` counter).

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

*Checks:* canary secrets in vault and fixtures · scanner over prompts, logs and audit in CI (`tests/canary`, SB-411) · laws: logs, prompts and audit rows, and secrets logged URL-escaped, base64 or hex, or under a credential-named attribute (SB-612) · Watch: the same scanner in production, findings are Sev-1 ([sb-observability(7)](sb-observability.7.md#canaries-and-the-secret-scanner)).

### LAW-08: At most one behind stable {#law-08}

No stable-channel tenant on the release train (custody `factory` or `tenant`) runs a release more than one behind the current stable. Detached cells are off the train ([sb-factory(7)](sb-factory.7.md)).

*Checks:* Model (`Rollout.tla: OneBehindMax`). Removing the promote guard produces a counterexample · release-controller check on every promotion · Watch: hourly lag.

### LAW-09: Data is not instructions {#law-09}

Ingested content can't cause a tool call or exfiltrate data. Only the user's own request can lead to a tool call.

*Checks:* Eval (`injection`, zero tool calls and zero canary leaks) · Property: tool-argument provenance never from flagged spans · the pen test's corpus, `security/attacks/injection.txt`: every payload is flagged or stripped (SB-612).

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

*Checks:* Model (`Handoff.tla: NoUnlistedAccess`) · Verify (`sb factory access`, which enumerates access paths from IaC state and IAM, sb-factory(1)) · Property: every hand-off run in the controller ends with only the retained grants (`TestLAW17_HandoffLeavesOnlyRetainedGrants`, red until SB-705) · Watch: monthly re-enumeration.

### LAW-18: The root key always has a custodian {#law-18}

At every moment, at least one custodian can unwrap a cell's root key. A hand-off re-wraps under the new custodian **before** destroying the old wrap.

*Checks:* Model (`Handoff.tla: KeyNeverOrphaned`) · the wrap inventory's ordering guard: the factory wrap can't be destroyed unless every data key's tenant wrap unwraps now (`TestLAW18_DestroyNeedsAWorkingWrap`, SB-704) · step-ordering property over random hand-offs, aborts and failures in the controller (`TestLAW18_HandoffNeverOrphansTheKey`, red until SB-704 and SB-705) · Watch: key-wrap inventory per cell.

### LAW-19: Spaces only narrow {#law-19}

No answer, citation, draft or action uses a chunk that the source's live ACL would deny, whatever any space policy says. A space can hide what someone could open, never show what they can't (ADR-0020, [sb-spaces(7)](sb-spaces.7.md)).

*Owner:* platform lead. *Checks:* Model (`Spaces.als: SpacesOnlyNarrow`) · Property over random companies with spaces, labels, barriers and ACLs (`TestLAW19_SpacesOnlyNarrow` at the policy layer since SB-1004; `TestLAW19_TheSpaceContextNeverWidens`) · Static (`make policy`: only the base policy permits read, and the whole set allows nothing it doesn't) · Eval (`space-leak`).

### LAW-20: Restricted stays home {#law-20}

Content labelled `restricted` reaches only members of a space that holds it, including in company-wide asks.

*Owner:* platform lead. *Checks:* Model (`Spaces.als: RestrictedStaysHome`) · Property (`TestLAW20_RestrictedStaysHome` on the ask path since SB-1006; `TestLAW20_IngestNeverLabelsBelowRestricted`) · Eval (`space-leak`, People, Finance and Executive cases).

### LAW-21: Moves revoke fast {#law-21}

When the directory removes someone from a space's member groups, they lose that space within 5 minutes. A new membership grants nothing until the cell has confirmed it from the directory.

*Owner:* identity lead. *Checks:* Property on a fake clock (`TestLAW21_MovesRevokeFast`, red until SB-1205) · Watch (move-to-revoke latency, W12).

### LAW-22: Barriers hold {#law-22}

No answer, citation, draft or action crosses a declared information barrier, in either direction.

*Owner:* platform lead. *Checks:* Model (`Spaces.als: BarriersHold`) · Property (`TestLAW22_BarriersHold`: asks, drafts, tool calls and approvals, with source owners; SB-1203) · Eval (`space-leak`).

### LAW-23: Grants are consented and revocable {#law-23}

A parent-org grant on a company space carries only the aggregates the company approved, never content, and stops within 5 minutes of being revoked (ADR-0014; built only if Q-01 is yes).

*Owner:* portfolio lead. *Checks:* Property (`TestLAW23_GrantsAreConsentedAndRevocable`, red until SB-1302).

### External review (SB-612)

Before the platform scales past friendly tenants, an independent team pen-tests it against staging, with test tenants only. The scope and rules of engagement are in `security/README.md`. Every finding goes in the register, `security/findings.yaml`, with its severity, the laws it touches, and (once fixed) its `BUG-nnnn` entry. A fixed finding is also named in the threat model. `make check` fails while any **critical** or **high** finding isn't fixed, so a release can't go out over one. Medium and lower findings may be `accepted` with a reason in the threat model's residual risks. When a finding shows that a law was violated, a law test that would have caught it is added before the fix, and it must go red first.

## SEE ALSO

[sb-factory(7)](sb-factory.7.md), [Assurance ledger](../living/ASSURANCE_LEDGER.md), [04 Testing and assurance](../docs/04-testing-and-assurance.md), [proofs/](../proofs/README.md)
