---
title: "SB-504: Reproducible builds, SBOM and signing"
summary: "We can only prove what ran if two builds agree."
---

# SB-504: Reproducible builds, SBOM and signing

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.2 Supply chain |
| Stories | US-06.2 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06) |
| Depends on | [SB-503](../E06-evals-release/SB-503-release-bundle-pins-prompts-tools-policies.md) |
| Status | Open |

## Why

We can only prove what ran if two builds agree.

## Acceptance criteria

- [ ] Two independent builds of a commit produce identical digests
- [ ] SBOM (SPDX) attached; artefacts signed

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Hermetic build flags
- [ ] SBOM generation
- [ ] Signing step
- [ ] Rebuild comparison job
- [ ] Write or extend the law tests: LAW-06
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
