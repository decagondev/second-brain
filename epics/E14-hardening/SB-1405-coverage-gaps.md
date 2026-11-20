---
title: "SB-1405: Close the coverage gaps in the core"
summary: "A package with thin tests is one whose next change can break silently."
---

# SB-1405: Close the coverage gaps in the core

| Field | Value |
|---|---|
| Epic | [E14 Hardening](README.md) |
| Wave / slice | W13 / S13.5 |
| Stories | US-14.1 |
| Manual | as the change needs (manual first, P2) |
| Laws | every law a thin package carries |
| Depends on | [SB-1404](SB-1404-manual-matches-code.md) |
| Status | Open |

## Why

A package with thin tests is one whose next change can break silently.

## Acceptance criteria

- [ ] A coverage report for every core package, and tests where a package is thin
- [ ] Every package under `internal/core` has a coverage figure in the report, and none is below the floor the report sets
- [ ] Every new test is a unit or law test that fails when the behaviour it covers is broken

## Subtasks

- [ ] Update the man pages the change touches first (manual first, P2)
- [ ] Coverage report
- [ ] Tests for thin packages
- [ ] A floor in `make check`
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
