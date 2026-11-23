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
| Status | Done |

## Why

`--check` checks a machine before an install; once the cell runs, IT has no single command that says what's wrong with it and how to fix it.

## Acceptance criteria

- [x] `install.sh --doctor CONFIG` checks the running box: services up, timers enabled and last run green, free disk, TLS certificate expiry, `SB_OFFSITE` mounted, writable and off the root disk, `/readyz`, and whether `build/selfhost` is on an encrypted disk (a warning)
- [x] Each line is `pass`, `warn` or `fail` with the command that fixes it, or `unknown` for a check that can't run, never `pass`; it exits 1 on any `fail` or `unknown` line
- [x] The thresholds and verdicts are a Go function with a test per check (`sb selfhost doctor` reads facts the script gathers), so a check can be made to fail in CI

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Facts gatherer in the script
- [x] Verdicts in Go with tests
- [x] Runbook Check steps call it
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.1): `install.sh --doctor` runs `tools/selfhost/facts.sh`, which gathers the box's facts as one JSON document (compose service states, both timers' enabled state and last result, disk use of the install directory and Docker's data root, the certificate's expiry from `build/selfhost/tls/cert.pem` or Caddy's own, `SB_OFFSITE`'s mount and a write probe as uid 65532, the newest dump and off-box copy, `/readyz` through the proxy, LUKS under the install directory) and reads no secret. `sb selfhost doctor` judges them in `internal/core/box`: pass, warn, fail or unknown, worst first, each non-pass line with its fix; it exits 1 on any fail or unknown (sb(1)). `tools/selfhost/sb.sh` runs the bundle's `bin/sb` (the bundle now carries it) or builds it. Run against the demo box here: it found the demo's off-box directory on the root disk (as designed) and that `/readyz` answers 204, which the first draft wrongly failed. Tests: a test per check that makes it fail, stale and future facts, and `TestLAW03_BoxNeverShowsAGapAsHealthy` with a self-test that catches a judge treating unknown as pass.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
