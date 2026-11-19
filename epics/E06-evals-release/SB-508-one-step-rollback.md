---
title: "SB-508: One-step rollback"
summary: "Rollback is one command and one audit row."
---

# SB-508: One-step rollback

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.3 Rings |
| Stories | US-06.3 |
| Manual | [`sb-release(1)`](../../manual/sb-release.1.md) |
| Laws | [LAW-08](../../manual/sb-laws.7.md#law-08) |
| Depends on | [SB-507](../E06-evals-release/SB-507-release-controller-with-channels-and-rings.md) |
| Status | In review |

## Why

Rollback is one command and one audit row.

## Acceptance criteria

- [ ] `sb release rollback` restores the previous stable in under 10 minutes (drill) (drill built and timed with a fake deployer; real cells with W7)
- [x] Rolled-back release is marked revoked

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Rollback command
- [x] Drill
- [x] Release notes entry (the drill report and the `kind: release` audit row)
- [x] Write or extend the law tests: LAW-08
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.3a): `State.Rollback` revokes stable, restores the previous stable and drains every cell on the revoked release, as in `Rollout.tla` (BUG-0001). `State.Drill` rolls back, redeploys each draining cell through a deployer, and times the whole thing against the 10-minute target. A cell whose redeploy fails stays draining and is named, never left serving the revoked release. `sb release rollback [--reason]` audits, then saves. A second rollback is refused, because there is no earlier stable, and a revoked release can't be promoted again.

**Left:** the drill against real cells (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
