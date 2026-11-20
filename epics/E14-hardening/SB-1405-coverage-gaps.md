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
| Status | Done |

## Why

A package with thin tests is one whose next change can break silently.

## Acceptance criteria

- [x] A coverage report for every core package, and tests where a package is thin
- [x] Every package under `internal/core` has a coverage figure in the report, and none is below the floor the report sets
- [x] Every new test is a unit or law test that fails when the behaviour it covers is broken

## Subtasks

- [x] Update the man pages the change touches first (manual first, P2)
- [x] Coverage report
- [x] Tests for thin packages
- [x] A floor in `make check`
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S13.5): coverage profiles from every Go suite (`make test`, `make laws`, the four game-day and canary suites) merged by `tools/check/coverage.py`, with a self-test, at the end of `make smoke`. Measured that way, one core package was below 70%: `proposals` (67.7%), whose atomic `Update` and copying store had no test of their own; it is at 100% now. `barrier` (74.5%) gained a test of its action and approval facts, refusals included. The floor is 75%; the lowest package today is `ask` at 77.9%. Measured by unit tests alone, `runtime`, `barrier`, `cell` and `retrieve` look thin (14% to 33%), because their laws test them; the merged figure is the honest one.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
