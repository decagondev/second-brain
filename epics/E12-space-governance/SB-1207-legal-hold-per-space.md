---
title: "SB-1207: Legal hold per space"
summary: "A legal hold must stop deletion in the scope of a matter, and only there."
---

# SB-1207: Legal hold per space

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.4 Harden |
| Stories | US-12.4 |
| Manual | `sb-spaces(7)` (new), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-611](../E07-rollout-ops/SB-611-retention-and-deletion.md) |
| Status | Done |

## Why

A legal hold must stop deletion in the scope of a matter, and only there.

## Acceptance criteria

- [x] A Legal steward can place a hold on a space or a scope; retention and space deletion skip held items
- [x] Holds and releases are audited; a held item can't be crypto-shredded

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Hold record
- [x] Retention integration
- [x] Audit rows
- [x] Write or extend the law tests: LAW-11, LAW-13
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.4a): sb-spaces(7) gained "Legal holds, exactly", sb-tenant(5) the holds paragraph, sb-admin(3) three endpoints, and sb-errors(7) `EHOLD`. `internal/core/holds` keeps holds on a space, on custodians, or both, places and releases them with a `kind: retention` audit row written first (nothing changes if it can't be), and answers retention's question conservatively: a record whose space or user its store doesn't know is held by any hold that could cover it. `retention.Store.Expire` now takes the active holds and reports what they kept; traces carry their space, feedback takes its trace's, conversations and log files (whose space isn't kept) stay while any hold could cover them. `Job.Erase` refuses a custodian, or anyone while a whole space is held. `spaces.SyncHeld` refuses a manifest that would remove a held space before changing anything, so its row stays and its key is never shredded. `api.Holds` serves Legal stewards (Cedar `place_hold` and `release_hold`, the stewards of `legal`-template spaces). `TestLAW13_LegalHoldsStopDeletion` checks the sweep, erasure and space removal against an oracle, and LAW-11's audit-first rule for placing and releasing; a sweep that ignores holds and a sync that removes a held space are caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
