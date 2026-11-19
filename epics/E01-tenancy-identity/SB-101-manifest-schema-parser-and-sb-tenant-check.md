---
title: "SB-101: Manifest schema, parser and `sb tenant check`"
summary: "The manifest is the one file a company edits; errors must point at the exact line."
---

# SB-101: Manifest schema, parser and `sb tenant check`

| Field | Value |
|---|---|
| Epic | [E01 Tenancy and Identity](README.md) |
| Wave / slice | W1 / S1.1 Manifest |
| Stories | US-01.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-004](../E00-foundations/SB-004-first-drafts-sb-1-sb-tenant-1-sb-tenant-5.md) |
| Status | In review |

## Why

The manifest is the one file a company edits; errors must point at the exact line.

## Acceptance criteria

- [x] Invalid manifests fail with `file:line:col: field: message`
- [x] Any `version:` or `platform:` field is rejected (ADR-0004)
- [x] Unknown fields are errors, not warnings
- [x] JSON Schema published from the same source as the Go types

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Define `manifest.Tenant` Go type and generate JSON Schema
- [x] YAML parser that keeps line/column positions
- [x] Validation rules from `sb-tenant(5)`
- [x] Golden tests for every error message
- [x] Update man page examples to run in CI
- [x] Write or extend the law tests: LAW-16
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: built. `internal/core/manifest` holds a strict YAML-subset parser that keeps line and column (`yaml.go`), the checker (`manifest.go`) and the JSON Schema (`schema.go`). All three are standard library only, so ADR-0016 still holds. One field table drives the checker, the required list and the schema, and a test keeps the `Tenant` json tags in step with it. The schema is published as `examples/tenants/tenant.schema.json`, and a test fails if it's stale. `sb tenant check` prints `FILE: ok`, or compiler-format errors and a count line, then exits 1 with `EMANIFEST`. `--json` is supported. Golden tests cover all 66 messages at their exact positions, and `invalid.yaml` prints `invalid.expected` byte for byte (unit test and `make examples`). LAW-16 has a law test for the manifest side; the drift side comes with SB-510. `make check` is green in WSL.

Manual changes, made first: sb-tenant(5) gains a SYNTAX section and exact rules (regions `eu`/`us`, vendor names, known workflows, duplicates, `_`-separated forbidden names, `apply` owning the existing-cell comparison). sb-tenant(1) gains output and JSON format. sb-tenancy(7) gains the region list.

2026-11-19: the user confirmed regions `eu` and `us` "for now" and gave the go-ahead.

**Left:** the first green CI run on a real push (the folder isn't a git repository yet).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
