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
| Status | Open |

## Why

A prompt change is a release, with an id.

## Acceptance criteria

- [ ] Bundle manifest lists code digest, prompt hashes, tool schemas, policy hashes and model id
- [ ] Any change to any of them produces a new release id

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Bundle format
- [ ] Bundler
- [ ] Audit rows reference release id
- [ ] Write or extend the law tests: LAW-06
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
