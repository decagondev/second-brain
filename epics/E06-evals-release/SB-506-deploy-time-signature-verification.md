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
| Status | Open |

## Why

Unsigned builds don't start.

## Acceptance criteria

- [ ] Cells refuse to start an image whose signature or SBOM doesn't verify
- [ ] Cells pull bundles and verify them; the factory needs no inbound network access to any cell

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Admission check in deploy
- [ ] Negative test with unsigned image
- [ ] Write or extend the law tests: LAW-06
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
