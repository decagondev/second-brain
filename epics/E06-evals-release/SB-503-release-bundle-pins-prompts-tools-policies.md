---
title: "SB-503: Release bundle pins prompts, tools, policies and model"
summary: "A prompt change is a release, with an id."
---

# SB-503: Release bundle pins prompts, tools, policies and model

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.1 Eval gate |
| Stories | US-06.2 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06) |
| Depends on | [SB-305](../E04-retrieval-answers/SB-305-model-gateway-adapter.md) |
| Status | In review |

## Why

A prompt change is a release, with an id.

## Acceptance criteria

- [x] Bundle manifest lists code digest, prompt hashes, tool schemas, policy hashes and model id
- [x] Any change to any of them produces a new release id

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Bundle format
- [x] Bundler
- [x] Audit rows reference release id
- [x] Write or extend the law tests: LAW-06 (`TestLAW06_OnlySignedGatedBuildsRun` written red; it goes green with signing and admission, SB-504 and SB-506)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.1a): `internal/core/release` defines `bundle.json`: artefact digests, prompt hashes, tool registry entries with their payload-schema hashes, platform policy and schema hashes, the pinned model, the gate file's hash and the build date. The id is `rel_<date>_<6 hex>` over the RFC 8785 canonical form, which now lives in `internal/core/canonjson` and is shared with approval payloads. A test changes each field in turn and gets a new id every time, and tool order doesn't matter. `Manifest.Check` catches a bundle whose id doesn't match its contents. `tools/release/bundle` builds the manifest from the repository (prompts, workflow templates, `policy/`, `release/model.json`, `evals/gate.yaml`) plus the built artefacts, and is tested against the real repository. Audit rows already carry the running release (`audit.Log.Release`). A cell passes the bundle id there.

**Left:** signing and verifying the bundle (SB-504, SB-506), after which LAW-06 goes green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
