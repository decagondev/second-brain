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
| Status | Open |

## Why

Operators, approvers and askers see different consoles.

## Acceptance criteria

- [ ] Roles map to Cedar policies
- [ ] No role can see another tenant

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Role policies
- [ ] UI gating
- [ ] Tests
- [ ] Write or extend the law tests: LAW-02
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
