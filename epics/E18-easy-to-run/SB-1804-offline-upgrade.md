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
| Status | Open |

## Why

An air-gapped site runs `docker load` on images nothing has checked, then reinstalls; a tampered bundle reaches the Docker daemon before the signature is looked at.

## Acceptance criteria

- [ ] `update.sh --bundle FILE` unpacks into a staging directory and checks the release's signature against `SB_RELEASE_TRUSTED` and every image tar's digest against the release before any `docker load`
- [ ] A bundle that fails any check changes nothing and loads nothing
- [ ] Then it copies the release into the release directory and runs the normal upgrade, so the updater admits it exactly as online

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Verifier (Go, `sb release verify-bundle`)
- [ ] Script
- [ ] Tests with a tampered tar and an unsigned release
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
