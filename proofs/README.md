---
title: Proofs
summary: TLA+ and Alloy models of approvals, rollout and permissions.
---

# Proofs

Formal models of the four designs where a subtle mistake would break a law. They come before the code (ADR-0010). State and field names match the man pages and the code, so drift is visible in review.

| Model | Tool | Laws | Ticket | Status |
|---|---|---|---|---|
| [tla/Approvals.tla](tla/Approvals.tla) | TLA+ / TLC | LAW-04, LAW-05, LAW-10, LAW-15 | SB-011 | TLC clean, 1,068 states; 4 of 4 mutations caught |
| [tla/Rollout.tla](tla/Rollout.tla) | TLA+ / TLC | LAW-06, LAW-08 | SB-012 | TLC clean, 6,290 states; 4 of 4 mutations caught |
| [alloy/Permissions.als](alloy/Permissions.als) | Alloy 6 | LAW-01, LAW-02, LAW-03 | SB-013 | 5 checks UNSAT at scope 6; mutation caught |
| [tla/Handoff.tla](tla/Handoff.tla) | TLA+ / TLC | LAW-17, LAW-18 | SB-705 | TLC clean, 23 states; 4 of 4 mutations caught |
| [crosscheck.py](crosscheck.py) | Python | mirrors the three TLA+ models | — | Runs clean; all mutations caught |

The models were written during design on 2026-11-18 and first run on 2026-11-19 (`make proofs`, locally; the first CI run is still to come). The ledger moves a law to *Modelled* only after a clean run **and** a mutation check. The mutation table below is automated in `tools/ci/mutations.py`, which `make proofs` runs after the models.

## Cross-check (until TLC runs in CI)

[`crosscheck.py`](crosscheck.py) is a hand-written explicit-state search that mirrors the TLA+ models in Python. It is **not** a substitute for TLC: it checks the design logic, not the TLA+ text. Result on 2026-11-18 (after the non-atomic rollback fix, BUG-0001):

| Model | Reachable states | Invariants | Mutations caught |
|---|---|---|---|
| Approvals | 1,068 | all hold, including `KillWins` | 5 of 5 |
| Rollout | 6,290 | all hold | 4 of 4 |
| Handoff | 23 | all hold | 4 of 4 |

```sh
python3 proofs/crosscheck.py
```

## Run them

```sh
make proofs        # fetches checksum-pinned jars (tools/versions.env), then runs tools/ci/proofs.sh
# which runs, per model, from proofs/tla (TLC needs the bare file name):
java -cp ../../tools/vendor/tla2tools.jar tlc2.TLC -workers auto -deadlock -config Approvals.cfg Approvals.tla
java -cp ../../tools/vendor/tla2tools.jar tlc2.TLC -workers auto -deadlock -config Rollout.cfg   Rollout.tla
java -cp ../../tools/vendor/tla2tools.jar tlc2.TLC -workers auto -deadlock -config Handoff.cfg   Handoff.tla
# and one Alloy call per check or run (exec only runs the default command otherwise):
java -jar tools/vendor/alloy.jar exec -f -c NoCrossTenant -t none -o OUT proofs/alloy/Permissions.als
# then the mutation checks:
python3 tools/ci/mutations.py
```

Tool jars are pinned by version and SHA-256 in `tools/versions.env` and fetched by `make tools`, never committed. `tools/ci/proofs.sh` passes a model only when the checker positively reports success; any other output fails (P1). CI runs it as the `proofs` job.

## What each model says

**Approvals.** One action moves through `Drafted → PendingApproval → Approved → Executing → Ran | DidNotRun`, with `Rejected`, `Expired` and edit-voids-approval on the side. The environment can also *tamper* with the stored payload after approval, and kill the agent at any moment. Invariants: `AuditBeforeExec`, `ApprovalBeforeExec`, `ExecutedMatchesApproved`, `NoSelfApproval`, `RanImpliesReceipt`, `AtMostOnce`, plus the action property `KillWins`.

**Rollout.** Releases are built and may or may not pass the gate. Testing tenants take gated releases; stable tenants follow stable; releases are promoted under the **promote guard**. Rollback and revocation are **not atomic**: revoked cells drain, then redeploy one by one. Invariants: `OnlyGatedRuns`, `RevokedOnlyWhileDraining`, `DrainingOnlyRevoked`, `OneBehindMax` and `StableNotAhead` (both for serving cells), `StableNotRevoked`.

**Handoff.** One cell moves from factory to tenant custody: plan, dual approval of the plan hash, provision, re-wrap the root key, move, verify, cut over, destroy the factory wrap, and optionally detach. Abort is possible any time before cutover. Invariants: `KeyNeverOrphaned`, `ServingCanUnwrap`, `NoUnlistedAccess`, `FactoryKeyGone`, `DualApproval`, `ManagedIsIntact`. Hand-back isn't modelled yet.

**Permissions.** Tenants, users, groups and documents with a possibly stale prefilter; a live verdict per candidate that may be `Allow`, `Deny`, `Unknown` or missing. Assertions: `NoCrossTenant`, `CitedImpliesLiveAllow`, `UnknownNeverUsed`, `MissingNeverUsed`, `StalePrefilterHarmless`.

## Mutation checks (all 13 caught on 2026-11-19)

| Mutation | Expected counterexample |
|---|---|
| Approvals: remove `u # requester` from `Approve` | `NoSelfApproval` |
| Approvals: remove `payload = approvedHash` from `Execute` | `AuditBeforeExec` or `ExecutedMatchesApproved` (via `Tamper`) |
| Approvals: remove `~killed` from `Execute` | `KillWins` |
| Approvals: remove the `Log` from `Execute` | `AuditBeforeExec` |
| Rollout: remove the promote guard | `OneBehindMax` after two promotions |
| Rollout: drop `r \in gated` from `Promote` | `OnlyGatedRuns` |
| Rollout: let a testing tenant deploy a revoked release | `RevokedOnlyWhileDraining` |
| Rollout: roll back without marking cells as draining | `RevokedOnlyWhileDraining` |
| Handoff: provision without both approvals | `DualApproval` |
| Handoff: destroy the factory wrap before the tenant wrap exists | `KeyNeverOrphaned` |
| Handoff: cut over without revoking factory paths | `NoUnlistedAccess` |
| Handoff: cut over before the tenant wrap exists | `ServingCanUnwrap` |
| Permissions: `context = candidates - live.Deny` | `UnknownNeverUsed`, `MissingNeverUsed` |
