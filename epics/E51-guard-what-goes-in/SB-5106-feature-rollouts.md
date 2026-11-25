---
title: "SB-5106: Feature rollouts"
summary: "A new feature turns on for everyone at once, with no pilot."
---

# SB-5106: Feature rollouts

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.2 |
| Stories | US-51.2 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md), [`sb-release(1)`](../../manual/sb-release.1.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-06, LAW-08 |
| Depends on | SB-5006, SB-012 |
| Status | Open |

## Why

A new feature turns on for everyone at once, with no pilot.

## Acceptance criteria

- [ ] A release lists its features with `rollout: pilot|on`; a pilot feature is on only for the tenant's `pilot` group
- [ ] After 14 days it is on for everyone unless an admin holds it (`sb rollout hold|release FEATURE`), audited
- [ ] The admin's week lists features in pilot, held and released

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Feature list in the release
- [ ] Pilot gate and hold
- [ ] Admin's week lines
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
