---
title: "SB-707: Releases after hand-off"
summary: "A handed-off cell stays patched by pulling from the train."
---

# SB-707: Releases after hand-off

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.2 Hand-off |
| Stories | US-08.4 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md), [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-08](../../manual/sb-laws.7.md#law-08) |
| Depends on | [SB-705](../E08-factory-handoff/SB-705-hand-off-workflow-with-dual-approval.md), [SB-507](../E06-evals-release/SB-507-release-controller-with-channels-and-rings.md) |
| Status | In review |

## Why

A handed-off cell stays patched by pulling from the train.

## Acceptance criteria

- [ ] Tenant-custody cell pulls and verifies bundles with only the `publish` grant (over HTTPS with a read-only pull token, against an in-process store; a cell running in a real tenant account is W7's runtime)
- [x] Promote guard counts tenant-custody cells

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Pull path test from a tenant account
- [x] Controller includes tenant-custody cells
- [x] Runbook for a tenant pausing pulls (becomes detach)
- [x] Write or extend the law tests: LAW-06, LAW-08
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.2d): sb-release(7) gained "Cells in a tenant's account". A tenant-custody cell pulls with a read-only token from its own vault. The `publish-releases` grant gives the factory nothing in the tenant's account. Tenant-custody cells count in the promote guard, a pause becomes a detach, and a detached cell leaves the train. `internal/adapters/release/httpstore` is the cell's HTTPS client for the store, and it refuses bad paths and objects over 256 MB. `rollout.Tenant` carries its custody. `SetCustody` records a hand-off, and `Detach` takes a tenant-custody cell off the train, so it is never counted or deployed to again. A unit test shows an un-pulled tenant-custody cell blocking promotion until it detaches. Two new law tests: `TestLAW06_TenantAccountPullsOnlySignedGatedBuilds` runs the admission law through the HTTPS pull path with a pull token, and `TestLAW08_TenantCustodyCellsStayOneBehind` runs the one-behind law with half the stable tenants in tenant custody. A self-test catches a guard that skips them. The runbook is `docs/runbooks/pause-pulls.md`.

**Left:** a cell actually running in a tenant's account and pulling from the factory's store (W7 runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
