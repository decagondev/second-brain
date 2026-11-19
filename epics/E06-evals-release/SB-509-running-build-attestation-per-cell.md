---
title: "SB-509: Running-build attestation per cell"
summary: "Check what is actually running, not what we think we deployed."
---

# SB-509: Running-build attestation per cell

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.3 Rings |
| Stories | US-06.2 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06) |
| Depends on | [SB-506](../E06-evals-release/SB-506-deploy-time-signature-verification.md) |
| Status | In review |

## Why

Check what is actually running, not what we think we deployed.

## Acceptance criteria

- [x] Each cell reports running digests hourly and on deploy; mismatch is Sev-1
- [x] Attestations are pushed by the cell, so the check works for cells in any account

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Attestation agent (signing is built: `attest.Sign`; the cell's hourly loop comes with the cell runtime, W7)
- [x] Controller comparison
- [x] Alert
- [x] Write or extend the law tests: LAW-06
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.3b, with SB-510): `internal/core/attest`. A cell signs `{tenant, release, digests, seq, at}` with its own Ed25519 attestation key, over the report's canonical form. `Checker.Accept` verifies the signature with that cell's key, refuses replays (`seq` must increase), and checks that the release is the one the controller deployed there and that every attested artefact is in that release's bundle with the same digest. Each failure is a named Sev-1 finding: `attestation_mismatch` or `attestation_replay`. `Silent` reports `attestation_missing` after 2 hours with no accepted report. Tests cover a good report, a replay, the wrong release, the wrong digest, an extra binary, a forged signature, a report edited after signing, and silence. LAW-06 is extended to what is running, not only what is admitted.

**Left:** the cell's attestation loop and the alert webhook wiring in a running cell (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
