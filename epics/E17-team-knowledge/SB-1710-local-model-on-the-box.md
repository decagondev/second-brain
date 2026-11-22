---
title: "SB-1710: A local model on the box"
summary: "A sovereign cell that sends every question to a hosted model isn't fully sovereign, and the hashing embedder retrieves poorly."
---

# SB-1710: A local model on the box

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.5 |
| Stories | US-17.5 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-02, LAW-06, LAW-07, LAW-13 |
| Depends on | [SB-1701](SB-1701-search-without-the-model.md) |
| Status | Proposed |

## Why

A sovereign cell that sends every question to a hosted model isn't fully sovereign, and the hashing embedder retrieves poorly.

## Acceptance criteria

- [ ] A compose profile `model` runs one OpenAI-compatible model server, pinned by digest, on the `data` network only, with no route out and no secret
- [ ] An `openaicompat` embedder behind the `Embedder` port; changing the embedder rebuilds the index from the sources
- [ ] The release bundle pins the model id and the weights' SHA-256, the eval gate runs on that exact model, and a model that fails it doesn't run
- [ ] The offline install bundle can carry the weights; `make selfhost-model` tries it; the runbook states the hardware and the latency to expect

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] ADR for the model server
- [ ] Compose profile and embedder adapter
- [ ] Bundle pinning and the gate
- [ ] Smoke checks
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Proposed 2026-11-22 with W16 ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
