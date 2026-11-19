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
| Status | In review |

## Why

We can only prove what ran if two builds agree.

## Acceptance criteria

- [x] Two independent builds of a commit produce identical digests
- [x] SBOM (SPDX) attached; artefacts signed

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Hermetic build flags
- [x] SBOM generation
- [x] Signing step
- [x] Rebuild comparison job
- [x] Write or extend the law tests: LAW-06
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.2a, with SB-506): `tools/ci/repro.sh` builds `sb` twice with `CGO_ENABLED=0 -trimpath -buildvcs=false`, an empty build id and separate fresh caches, and compares digests (identical locally). `tools/release/sbom` writes SPDX 2.3 JSON from the binary's build info, using `go.sum` hashes for vendored modules, plus every package in `web/package-lock.json` with its integrity hash, all with unique SPDX ids. `make release-build` runs the repro check, the SBOM and the bundle, with the SBOM as a bundle artefact, and writes the message to sign. `.github/workflows/release.yml` runs it on every push to `main`, signs with cosign keyless (GitHub OIDC; `sigstore/cosign-installer` pinned by commit), verifies the signature against this workflow's identity, and uploads the signed set. Development signing is Ed25519 (`release.Ed25519`).

**Left:** confirm the first keyless signature in CI, and a second independent build machine for the rebuild comparison (the two builds share a runner today).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
