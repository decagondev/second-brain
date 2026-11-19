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
| Status | Open |

## Why

Check what is actually running, not what we think we deployed.

## Acceptance criteria

- [ ] Each cell reports running digests hourly and on deploy; mismatch is Sev-1
- [ ] Attestations are pushed by the cell, so the check works for cells in any account

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Attestation agent
- [ ] Controller comparison
- [ ] Alert
- [ ] Write or extend the law tests: LAW-06
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
