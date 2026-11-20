---
title: "SB-1201: Stewards and delegated space changes"
summary: "Department leads know their space best; IT shouldn't be the bottleneck, and leads mustn't be able to widen access."
---

# SB-1201: Stewards and delegated space changes

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.1 Stewards |
| Stories | US-12.1 |
| Manual | `sb-spaces(7)` (new), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | [LAW-15](../../manual/sb-laws.7.md#law-15), [LAW-16](../../manual/sb-laws.7.md#law-16), LAW-19 (new, W8) |
| Depends on | [SB-1001](../E10-department-spaces/SB-1001-spaces-in-the-manifest-and-sb-tenant-check.md), [SB-602](../E07-rollout-ops/SB-602-admin-console-where-every-action-is-a.md) |
| Status | In review |

## Why

Department leads know their space best; IT shouldn't be the bottleneck, and leads mustn't be able to widen access.

## Acceptance criteria

- [x] A steward proposes a change to their own space in the console as a manifest diff (the API the console calls; its screens come with the console work)
- [x] The tenant operator approves the exact diff (hash-bound, LAW-15); the steward can't approve their own
- [x] A diff that would permit past the source or remove a barrier is refused at check time

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Steward role in Cedar
- [ ] Console flow (the API is built; the console's screens are left)
- [x] Check-time refusals
- [x] Write or extend the law tests: LAW-15, LAW-16, LAW-19
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.1a): sb-spaces(7) gained "Delegated changes, exactly", sb-admin(3) "Proposals" and four endpoints, and the schema and base policy the `propose_space` and `approve_space` actions: stewards of a space (its `stewards` groups, passed as the asker's groups) may propose; operators may approve; a forbid stops anyone approving their own. `manifest.StewardChange` replaces one space's definition and refuses a change of template, a lower label, no stewards, an unknown field (so a proposal can't carry barriers), or anything outside the space; `manifest.WithSpace` fills unset fields as the checker does and runs the full check. `internal/core/proposals` keeps each proposal with its base and proposed manifest hashes. `api.Stewards` serves the flow: propose (audited, nothing changes), list, approve with the exact hash against an unchanged base (`EINVAL` for a wrong hash, `EDRIFT` and a stale proposal for a changed base), and reject with a reason; every decision is a `kind: config` audit row with the space. The new law test `TestLAW15_StewardsNeverApproveTheirOwn` drives the real handler and policies with stewards, operators and a steward who is also an operator, and checks LAW-15, LAW-16's exact-manifest approval and LAW-19's within-the-space changes; policies without the self-approval forbid, and a rule that lets any change through, are caught, and the test fails if no proposal was ever approved.

**Left:** the console's screens for proposing and approving.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
