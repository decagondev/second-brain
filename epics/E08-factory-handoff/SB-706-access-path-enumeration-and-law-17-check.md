---
title: "SB-706: Access-path enumeration and LAW-17 check"
summary: "Prove the factory kept only what was agreed."
---

# SB-706: Access-path enumeration and LAW-17 check

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.2 Hand-off |
| Stories | US-08.3 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | [LAW-17](../../manual/sb-laws.7.md#law-17) |
| Depends on | [SB-705](../E08-factory-handoff/SB-705-hand-off-workflow-with-dual-approval.md) |
| Status | In review |

## Why

Prove the factory kept only what was agreed.

## Acceptance criteria

- [ ] `sb tenant verify --law LAW-17` enumerates factory access paths from IaC state and IAM and fails on any not in the hand-off record (the check, its GCP IAM source and `sb factory access` are built; `verify` against a deployed cell and real IAM is W7)
- [ ] Monthly re-check alerts on drift (`access.Watch` sets the gauge the LAW-17 Sev-1 reads; scheduling it monthly needs the control-plane runtime)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Enumerator
- [x] Comparison with hand-off record
- [ ] Monthly job (the function is built; the schedule is the runtime's)
- [x] Alert
- [x] Write or extend the law tests: LAW-17
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.2c): sb-factory(7) gained "Access paths, exactly", and sb-factory(1) and sb-tenant(1) follow it. `internal/core/access` has `Check`, which enumerates every source and reports each path no retained grant serves. A source that can't be read is `EUNAVAIL`, never a pass. It also has the wrap-inventory source and `Watch`, which sets `sb_unlisted_access_paths`, and to 1 when the check can't finish, feeding the LAW-17 Sev-1 rule from SB-701. The `gcp.IAM` source reads the project's, the deploy account's and the cell key's IAM policies with a separate read-only verifier identity (`gcp.VerifierPermissions`), and maps `sbPlatformKill` and `monitoring.viewer` to their grants. `gcptest` now serves `getIamPolicy`. `sb factory access TENANT` works on the local control plane, and the smoke test runs it. The new law test `TestLAW17_AccessCheckFindsEveryUnlistedPath` uses random factory and tenant bindings and key wraps against a model, and shows a failed read never passes. A self-test catches a source that skips the key's policy.

**Left:** `sb tenant verify --law LAW-17` against a deployed cell and real IAM, and the monthly schedule, both in the control-plane runtime.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
