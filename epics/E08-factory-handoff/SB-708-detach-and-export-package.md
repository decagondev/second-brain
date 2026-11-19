---
title: "SB-708: Detach and export package"
summary: "A divestiture must not strand a company's knowledge."
---

# SB-708: Detach and export package

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.3 Exit paths |
| Stories | US-08.5 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md), [`sb-audit(5)`](../../manual/sb-audit.5.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-17](../../manual/sb-laws.7.md#law-17) |
| Depends on | [SB-705](../E08-factory-handoff/SB-705-hand-off-workflow-with-dual-approval.md) |
| Status | In review |

## Why

A divestiture must not strand a company's knowledge.

## Acceptance criteria

- [x] Export contains manifest, policies, evals, signed audit export with anchor key, bundle and SBOM, runbooks
- [ ] After detach, the factory holds zero access paths and keeps only custody history (the controller's Detach revokes every path, checked by the LAW-17 hand-off law test, and the registry tombstone keeps only history; `sb factory detach` on a running cell is W7's runtime)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Export builder
- [x] Offline verification of the export
- [x] Registry tombstone
- [x] Write or extend the law tests: LAW-11, LAW-17
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S7.3a): sb-factory(7) gained "Detach, exactly", with the package layout, the offline check and the five-step order. sb-factory(1) gained `verify-export`, and sb-audit(5) notes the anchor key that ships with the export. `internal/core/export` writes a package with `MANIFEST.json` last and refuses a package missing any required item. `Verify` checks it offline, reusing `audit.Verify` with the pinned anchor key from `seq` 1, and `release.Admit` over the package's own release files with the release key in the package, rather than adding new checks. The registry (`localfs`) keeps an append-only custody history (`SetCustody`, which also implements the hand-off controller's Custody port). `Tombstone` removes a detached cell's record so it stops routing and keeps only that history. `sb factory verify-export DIR` maps failures to `ECHAIN`, `EUNSIGNED` and `ELAW`. The new law test `TestLAW11_ExportPackageVerifiesOffline` builds real packages and shows six kinds of tampering caught: a byte, a missing file, an extra file, a cut audit export, a swapped anchor key and an altered bundle. A self-test catches a digest-only verifier.

**Left:** `sb factory detach` against a running tenant-custody cell (W7 runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
