---
title: "SB-1804: Offline upgrade, verified before it's loaded"
summary: "An air-gapped site runs `docker load` on images nothing has checked, then reinstalls; a tampered bundle reaches the Docker daemon before the signature is looked at."
---

# SB-1804: Offline upgrade, verified before it's loaded

| Field | Value |
|---|---|
| Epic | [E18 Easy to Run](README.md) |
| Wave / slice | W17 / S17.2 |
| Stories | US-18.2 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-06 |
| Depends on | — |
| Status | Done |

## Why

An air-gapped site runs `docker load` on images nothing has checked, then reinstalls; a tampered bundle reaches the Docker daemon before the signature is looked at.

## Acceptance criteria

- [x] `update.sh --bundle FILE` unpacks into a staging directory and checks the release's signature against `SB_RELEASE_TRUSTED` and every image tar's digest against the release before any `docker load`
- [x] A bundle that fails any check changes nothing and loads nothing
- [x] Then it copies the release into the release directory and runs the normal upgrade, so the updater admits it exactly as online

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Verifier (Go, `sb release verify-bundle`)
- [x] Script
- [x] Tests with a tampered tar and an unsigned release
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S17.2): `update.sh --bundle FILE` unpacks into `build/selfhost/staging` (0700, removed on exit) and runs the installed `sb selfhost verify-bundle DIR --trusted KEY` (or `--cosign-identity`) with the cell's own trust: `release.Admit` on the bundle's `releases/` channel, then `release.VerifyImageTar` for every `image/NAME` the release names: every blob hashes to its name, `index.json` names exactly that image, everything it reaches is present, and nothing else (no links, no other paths) is in the archive. Only then are the verified archives loaded and the release copied into the cell's store, and the normal upgrade admits it again. `SB_BUNDLE_RELEASES=DIR make selfhost-bundle` carries the release. Run on the demo box here: a bundle of new images upgraded the box; the same bundle with 10 bytes of one layer changed was refused (`EUNSIGNED`) and nothing was loaded. Tests: `TestVerifyImageTar` (a changed or missing layer, another image, a second image in the index, no index, a symlink, a path outside the layout, `manifest.json` naming another blob), `TestSelfhostVerifyBundle` (another signer, a changed image, no release), `TestLAW06_OfflineBundlesLoadOnlyTheReleasesImages` with a self-test that catches an index-only check.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
