---
title: "SB-701: Final `sb-factory(7)` and the hand-off runbook"
summary: "Manual first: the hand-off is specified before it's built."
---

# SB-701: Final `sb-factory(7)` and the hand-off runbook

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.1 Custody model |
| Stories | US-08.1 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-604](../E07-rollout-ops/SB-604-sb-onboarding-7-day-0-to-90-runbook.md) |
| Status | In review |

## Why

Manual first: the hand-off is specified before it's built.

## Acceptance criteria

- [x] Every `sb factory` command has synopsis, options, exit codes and examples
- [ ] Runbook for tenant owners and factory operators, reviewed by security (written: `docs/runbooks/handoff.md`; the security review is human)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Finalise command grammar
- [x] Write runbook
- [ ] Review with a friendly tenant's IT (human)
- [x] Write or extend the law tests: LAW-17, LAW-18
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.1a): the new man page `sb-factory(1)` covers every command (`status`, `handoff plan|show|approve|execute|abort`, `access`, `detach`) with its synopsis, roles, options, exit codes and examples. sb-factory(7) is final: "The hand-off, exactly" gives the phases of `Handoff.tla` with what serves, which key wraps exist and which factory paths remain in each, the three invariants, abort, and the hand-off record. The grants a hand-off can retain are fixed (`publish-releases`, `platform-kill`, `support-metrics`), and none of them reads content or unwraps a key. The factory approval must come from an operator other than the planner. `docs/runbooks/handoff.md` covers before, plan and approve, execute, after and abort, with an expected result for every step. Two new law tests drive random hand-offs (plan, approvals including the planner's and stale ones, steps, failures, abort, replan, detach) and check the model's invariants after every operation. `TestLAW17_HandoffLeavesOnlyRetainedGrants` checks unlisted access, dual approval and an intact abort. `TestLAW18_HandoffNeverOrphansTheKey` checks that the key always has a custodian and that the serving cell can unwrap it. Both are red in the ratchet until SB-705. A reference controller written from the model passes both, and four broken ones are caught: a key destroyed early, paths kept, the planner approving, and abort keeping the tenant wrap.

**Left:** the security review of the runbook, and a friendly tenant's IT reading it (human).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
