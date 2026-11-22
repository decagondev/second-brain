---
title: "SB-1801: install.sh --doctor: check the running box"
summary: "`--check` checks a machine before an install; once the cell runs, IT has no single command that says what's wrong with it and how to fix it."
---

# SB-1801: `install.sh --doctor`: check the running box

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.1 |
| Stories | US-18.1 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-03, LAW-07 |
| Depends on | — |
| Status | Open |

## Why

`--check` checks a machine before an install; once the cell runs, IT has no single command that says what's wrong with it and how to fix it.

## Acceptance criteria

- [ ] `install.sh --doctor CONFIG` checks the running box: services up, timers enabled and last run green, free disk, TLS certificate expiry, `SB_OFFSITE` mounted, writable and off the root disk, `/readyz`, and whether `build/selfhost` is on an encrypted disk (a warning)
- [ ] Each line is `pass`, `warn` or `fail` with the command that fixes it; the exit code is the worst line, and a check that can't run is `fail`, never `pass`
- [ ] The thresholds and verdicts are a Go function with a test per check (`sb selfhost doctor` reads facts the script gathers), so a check can be made to fail in CI

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Facts gatherer in the script
- [ ] Verdicts in Go with tests
- [ ] Runbook Check steps call it
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
