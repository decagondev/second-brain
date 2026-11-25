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
| Status | Done |

## Why

A new feature turns on for everyone at once, with no pilot.

## Acceptance criteria

- [x] A release lists its features with `rollout: pilot|on`; a pilot feature is on only for the tenant's `pilot` group
- [x] After 14 days it is on for everyone unless an admin holds it (`sb rollout hold|release FEATURE`), audited
- [x] The admin's week lists features in pilot, held and released

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Feature list in the release
- [x] Pilot gate and hold
- [x] Admin's week lines
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `internal/core/features`: the release's `List` (this release: `branch-moderators` and `pause-link`, both `pilot`), `State.Seen` records when the cell first ran each, `State.For` says a feature is on for a person: for the `pilot` IdP group while in pilot or held, for everyone 14 days after it was first run unless held. The cell keeps the state in `state/features/state.json`. `GET /v1/admin/rollouts`, `POST /v1/admin/rollouts/FEATURE/hold|release`, and `sb rollout list|hold|release` through `GET|POST /v1/cell/rollouts…` for the control plane; each hold or release writes a `config` row `rollout:FEATURE` first. The admin's week has a Rollouts section. Tested in TestPilotHoldAndRelease and TestRollOutWithCareThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
