---
title: "SB-704: Root-key re-wrap and key custody transfer"
summary: "Custody of data is custody of keys."
---

# SB-704: Root-key re-wrap and key custody transfer

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.2 Hand-off |
| Stories | US-08.2 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-703](../E08-factory-handoff/SB-703-placement-adapter-for-tenant-owned-cloud.md) |
| Status | Done |

## Why

Custody of data is custody of keys.

## Acceptance criteria

- [x] Root key re-wrapped under tenant KMS before the factory wrap is destroyed
- [x] Key-wrap inventory per cell exposed to `sb factory status`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Re-wrap operation
- [x] Wrap inventory
- [x] Ordering test that fails if destroy precedes wrap
- [x] Write or extend the law tests: LAW-18
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.2a): sb-factory(7) gained "Key wraps, exactly". `internal/core/keywrap` has the wrap inventory: one wrap per data-key purpose per custodian. `Seed` makes a data key, and `apply` now makes the cell's `audit-signing` data key under the factory's root key when it stamps a cell. `Rewrap` is verified and idempotent. `Destroy` refuses with `ELAW` unless the kept custodian unwraps every data key at that moment. `Abort` is the same guard with the roles swapped, and `Inventory` checks every wrap live. `sb factory status` prints `key_wraps`, marks broken wraps, and exits 1 if a data key has no working wrap. The local control plane keeps the inventory in `DIR/keywraps` (`internal/adapters/keywrap/localfs`). `TestDestroyBeforeRewrapIsRefused` is the ordering test. The new law test `TestLAW18_DestroyNeedsAWorkingWrap` runs random seed, re-wrap, destroy, abort and corruption steps over two custodians' key stores. It went red on its first run, which caught an abort that counted wraps instead of checking them, and that is fixed. Self-tests catch an unchecked destroy and a drop of the last wrap.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
