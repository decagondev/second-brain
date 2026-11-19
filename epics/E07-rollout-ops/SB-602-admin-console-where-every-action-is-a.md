---
title: "SB-602: Admin console where every action is a manifest diff"
summary: "One source of truth, two editors."
---

# SB-602: Admin console where every action is a manifest diff

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.1 Verify and console |
| Stories | US-07.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-104](../E01-tenancy-identity/SB-104-sb-tenant-apply-idempotent-cell-creation.md), [SB-403](../E05-actions-approvals-audit/SB-403-kill-switches-per-agent-connector-and-tenant.md) |
| Status | In review |

## Why

One source of truth, two editors.

## Acceptance criteria

- [ ] Connectors, agents, approval rules and kill switches editable (all of them through the diff API; the console UI covers kill switches and make-permanent so far)
- [x] Each change produces a manifest diff and an audit row
- [x] Kill switches act immediately and offer 'make permanent' as a manifest diff

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Console UI
- [x] Admin API
- [x] Manifest writer
- [x] Audit hooks
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.1b, with SB-603): `manifest.Format` writes a valid tenant back as manifest YAML, and a round-trip test shows `Check(Format(t)) == t` for the examples. `manifest.Patch` applies JSON Patch on the console's paths only (never `tenant`, `region` or `idp`), runs the result through the same checker as `sb tenant check`, and `Diff` gives the line diff. `internal/api.Admin` serves the manifest, the diff (which changes nothing and writes a `kind: config` audit row with the patch's hash), kill, unkill, the kill list, and audit paging with a kind filter. `usage` and `refusals` say they arrive with SB-606 and SB-607. Every endpoint needs the operator role through the real cedar policies. The web console (`/?admin=1`) kills with a required reason, lists switches, clears them, and turns make-permanent into the manifest diff to apply. It is axe-clean.

**Left:** opening the diff as a pull request on the tenant's manifest repository, or signing and applying it (the cell runtime and repo integration), and console screens for connectors and approval rules.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
