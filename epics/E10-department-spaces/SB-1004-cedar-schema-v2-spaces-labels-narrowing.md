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
| Status | Done |

## Why

Space rules live in the one policy engine, and analysis must prove they never widen access.

## Acceptance criteria

- [x] Schema gains `Space` and labels; base policies add space and label forbids
- [x] Policy analysis proves for every shipped and tenant policy that the allowed set is a subset of the source-allowed set
- [x] LAW-19 green

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Schema
- [x] Base policies
- [x] Narrowing analysis in `make policy`
- [x] LAW-19 law test green
- [x] Write or extend the law tests: LAW-03, LAW-19
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S10.2a): sb-policy(5)'s spaces section is no longer a draft: schema v2, the four rules in `policy/platform/spaces.cedar`, and CI rule 5. sb-permissions(7) and sb-laws(7) point to them. Schema v2 adds `label` and `spaces` to `Chunk` and `space`, `member_spaces` and `barred_spaces` to the `read` context, all optional, so a cell without spaces is unchanged. Spaces are facts the kernel sets (`spaces.ReadContext`, `spaces.ChunkAttrs`), not entities, so no rule looks one up. The four forbids: a missing fact refuses; an ask in a space reads only what that space holds, for its members; only `public` and `internal` reach non-members of the holders (an unknown label is `restricted`); nothing held across a barrier. The Cedar adapter passes string sets. `tools/check/policy.py` adds rule 5: only `base.cedar` may permit `read` (an any-action permit counts), and over 1,728 space contexts and chunks the whole set allows nothing the base read permit doesn't. `TestLAW19_SpacesOnlyNarrow` now runs against the real kernel and policies with the source ACL last, and left the ratchet; a mutation that lets a space's allow skip the source ACL is caught. A self-test shows the policies agree with the reference cell on every ask and already hold LAW-20 and LAW-22 at the policy layer; those laws' tests stay on the ask path (SB-1006, SB-1203). `TestLAW03_MissingSpaceFactsRefuse` drops one fact at a time, and catches the policies without their missing-fact rule.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
