---
title: "SB-506: Deploy-time signature verification"
summary: "Unsigned builds don't start."
---

# SB-506: Deploy-time signature verification

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.2 Supply chain |
| Stories | US-06.2 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06) |
| Depends on | [SB-504](../E06-evals-release/SB-504-reproducible-builds-sbom-and-signing.md) |
| Status | In review |

## Why

Unsigned builds don't start.

## Acceptance criteria

- [x] Cells refuse to start an image whose signature or SBOM doesn't verify
- [ ] Cells pull bundles and verify them; the factory needs no inbound network access to any cell (admission pulls through `release.Store`; the cell runtime's pull loop comes with W7)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Admission check in deploy
- [x] Negative test with unsigned image
- [x] Write or extend the law tests: LAW-06
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.2a): `release.Admit` pulls a release and checks, in order: the id matches the contents, the bundle signature verifies for the trusted identity, every artefact to run (the SBOM included) matches its listed digest, and a signed gate record for this id says passed. Any failure is `EUNSIGNED`. A signature has to be made by the trusted identity: one that brings its own key proves nothing. There are two verifiers: Ed25519 (development) and `internal/adapters/release/cosign`, which runs `cosign verify-blob` pinned to one workflow identity and the GitHub issuer, refuses if cosign is missing, and is tested with a fake cosign. `internal/adapters/release/localfs` is a read-only-to-cells release store. Admission tests refuse each of: unsigned, someone else's key, an edited bundle, a swapped artefact, no gate, a failed gate, a gate for another release, an unsigned gate edit, and an unlisted artefact. `TestLAW06_OnlySignedGatedBuildsRun`, red since S5.1a, is green, and a cell that ignores the gate is caught.

**Left:** the cell runtime's pull-and-admit loop and the Sev-1 alert on refusal (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
