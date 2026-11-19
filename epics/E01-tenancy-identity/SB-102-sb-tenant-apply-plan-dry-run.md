---
title: "SB-102: `sb tenant apply --plan` dry run"
summary: "Operators should see the diff before anything changes."
---

# SB-102: `sb tenant apply --plan` dry run

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.1 Manifest |
| Stories | US-01.1 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-101](../E01-tenancy-identity/SB-101-manifest-schema-parser-and-sb-tenant-check.md) |
| Status | In review |

## Why

Operators should see the diff before anything changes.

## Acceptance criteria

- [x] `--plan` prints a resource-level diff and exits 0 without changes
- [x] Plan output is stable and machine-readable with `--json`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Compute desired state from manifest
- [x] Read actual state from control-plane registry (port `ports.Registry`; local `file://` adapter until SB-105)
- [x] Render human and JSON plan
- [x] Golden tests
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built. `internal/core/plan` turns a manifest into resources (cell, settings, connectors, excludes, agents, workflows) and diffs them against the applied state. `ports.Registry` is the new port. `internal/adapters/registry/localfs` serves `--endpoint file:///DIR` until the control plane exists (SB-105). It answers "no cell" only when the registry directory exists and has no record; anything else is `EUNAVAIL`. A region change fails with `EMANIFEST`. Apply without `--plan` is still `ENOTIMPL` (SB-104). Golden tests cover both sb-tenant(1) examples byte for byte, every line shape, stable JSON, and a pilot-to-full smoke test through the real binary. LAW-16 gains a plan-side law test.

**Left:** review by the user, and the first green CI run.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
