---
title: "SB-1808: A support bundle with no content"
summary: "When something breaks, IT sends screenshots and log snippets; nobody is sure what's safe to send."
---

# SB-1808: A support bundle with no content

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.3 |
| Stories | US-18.3 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | LAW-07 |
| Depends on | SB-1801 |
| Status | Done |

## Why

When something breaks, IT sends screenshots and log snippets; nobody is sure what's safe to send.

## Acceptance criteria

- [x] `install.sh --support` writes one tar: versions and digests, `--doctor` output, `ps`, and the last 1,000 log lines per service
- [x] Settings that name files are kept as names only; nothing under `/etc/second-brain` or `build/selfhost/vault` is read
- [x] Before it's written, the tar's text is scanned with the secret scanner (SB-411); any hit aborts the bundle

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Script
- [x] Scanner as `sb secrets scan`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.3): `install.sh --support` (`tools/selfhost/support.sh`) writes one tar (0600): versions and every running container's image id, `compose.env` and `cell.json` (names, paths, digests and vault references), the drill record and hold, the facts and the doctor's verdicts, `ps`, and the last 1,000 log lines of each service. Nothing under `/etc/second-brain` or the vaults is copied. Before it is written, `sb secrets scan --known build/selfhost/vault --known …` scans every staged file: `canary.Scanner.Learn` knows every stored value, each line of a multi-line one and the secret fields of a JSON one, and `ScanAll` adds the shapes (a private key, a URL with a password, a bearer token); any finding names file and line, never the value, exits `ELAW` and writes nothing. Run on the demo box: 14 files, 18 stored values, no finding; the bundle holds what the ticket lists. Tests: `TestLearnAndScanAll`, `TestSecretsScan`, `TestLAW07_SupportBundlesCarryNoSecret` with a self-test that catches a scanner that only knows whole values.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
