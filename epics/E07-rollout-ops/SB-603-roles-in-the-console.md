---
title: "SB-603: Roles in the console"
summary: "Operators, approvers and askers see different consoles."
---

# SB-603: Roles in the console

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.1 Verify and console |
| Stories | US-07.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-policy(5)`](../../manual/sb-policy.5.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02) |
| Depends on | [SB-602](../E07-rollout-ops/SB-602-admin-console-where-every-action-is-a.md) |
| Status | Done |

## Why

Operators, approvers and askers see different consoles.

## Acceptance criteria

- [x] Roles map to Cedar policies
- [x] No role can see another tenant

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Role policies
- [x] UI gating
- [x] Tests
- [x] Write or extend the law tests: LAW-02
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.1b): the operator role is a base Cedar permit: `admin` on the tenant's own `Cell` for a principal in `Group::"sb-operators"`, with the tenant condition and the existing cross-tenant forbid. Approvers come from the workflow approval groups (sb-approvals(7)). Cedar tests show that an operator administers their own cell and not another tenant's, and a member without the group gets `EPERM` on every admin endpoint. The API enforces it, and the console simply shows what the API returns. The policy-level LAW-02 law test (random principals, groups and resources) stays green with the new permit.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
