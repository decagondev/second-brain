---
title: "SB-1206: Reorg preview: who gains and loses before it happens"
summary: "Reorgs are planned; HR and IT should see the access effect before the IdP changes."
---

# SB-1206: Reorg preview: who gains and loses before it happens

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.3 Reorgs |
| Stories | US-12.3 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md), `sb-spaces(7)` (new) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-1205](SB-1205-scim-attributes-and-five-minute-moves.md) |
| Status | Done |

## Why

Reorgs are planned; HR and IT should see the access effect before the IdP changes.

## Acceptance criteria

- [x] `sb space diff --idp-snapshot FILE` lists, per person, the spaces they'd gain and lose
- [x] Flags anyone who would lose all spaces, or gain a restricted one
- [x] Works offline on a snapshot; no change is made

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] CLI
- [x] Snapshot format
- [x] Flags
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.3b): sb-space(1) gained `diff` and the snapshot format, sb-spaces(7) "Reorg preview", and sb(1) the synopsis line. `internal/core/reorg` parses a version-1 snapshot strictly (each person's groups before and after) and previews it against the manifest: per person, the spaces gained and lost, flagged `loses-all`, `gains-restricted`, or `barrier` (on both sides of one of the manifest's barriers after the move); people with no change are counted. `sb space diff FILE --idp-snapshot SNAPSHOT [--json]` runs it offline and writes nothing; the CLI test checks both files are byte-for-byte unchanged. `TestLAW16_ReorgPreviewIsExactAndChangesNothing` compares every preview with an independent oracle and checks the inputs are untouched; a preview without `loses-all`, one that reads only the first group, and one that rewrites its input are caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
