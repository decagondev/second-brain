---
title: "SB-1004: Cedar schema v2: spaces and labels, narrowing only"
summary: "Space rules live in the one policy engine, and analysis must prove they never widen access."
---

# SB-1004: Cedar schema v2: spaces and labels, narrowing only

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.2 Space policy |
| Stories | US-10.2 |
| Manual | [`sb-policy(5)`](../../manual/sb-policy.5.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md) |
| Laws | [LAW-03](../../manual/sb-laws.7.md#law-03), LAW-19 (new, W8) |
| Depends on | [SB-1002](SB-1002-space-registry-and-space-scoped-request-context.md), [SB-110](../E02-policy/SB-110-cedar-schema-and-base-policies.md) |
| Status | Open |

## Why

Space rules live in the one policy engine, and analysis must prove they never widen access.

## Acceptance criteria

- [ ] Schema gains `Space` and labels; base policies add space and label forbids
- [ ] Policy analysis proves for every shipped and tenant policy that the allowed set is a subset of the source-allowed set
- [ ] LAW-19 green

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Schema
- [ ] Base policies
- [ ] Narrowing analysis in `make policy`
- [ ] LAW-19 law test green
- [ ] Write or extend the law tests: LAW-03, LAW-19
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
