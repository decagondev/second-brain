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
| Status | In review |

## Why

A sovereign cell that sends every question to a hosted model isn't fully sovereign, and the hashing embedder retrieves poorly.

## Acceptance criteria

- [x] A compose profile `model` runs one OpenAI-compatible model server, pinned by digest, on the `data` network only, with no route out and no secret
- [x] An `openaicompat` embedder behind the `Embedder` port; changing the embedder rebuilds the index from the sources
- [x] The release bundle pins the model id and the weights' SHA-256, the eval gate runs on that exact model, and a model that fails it doesn't run
- [ ] The offline install bundle can carry the weights; `make selfhost-model` tries it; the runbook states the hardware and the latency to expect

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] ADR for the model server
- [x] Compose profile and embedder adapter
- [x] Bundle pinning and the gate
- [x] Smoke checks
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.5): built and tested without a model download. `embed` in sb-cell(5) and an `openaicompat` embedder (`internal/adapters/embed/openaicompat`, no key, every vector checked for size and place); `model.local` lets an `openaicompat` model on the box run with no key and a price of 0. Changing the embedder rebuilds the index: the ingest service records the embedder under state and, when it differs, drops the index and resets every connector's cursor; the Postgres index resizes its vector column when it does (`TestLAW13_EmbedderChangeRebuildsTheIndex` on a running cell). The release bundle pins `weights_sha256` and `embedder` (`tools/release/bundle --model-weights`), and lists `note-publish` with its schema. The compose profile `model` runs `model` and `model-embed` from one llama.cpp server image pinned by digest, on the `data` network only, read-only, as 65532, with no port and no secret; smoke checks look for each. `make selfhost-model` (`tools/selfhost/model.sh`) pins the image, hashes the weights, runs the gate's suites on that exact model through a loopback port it removes (`tools/evalcell` takes `openaicompat=BASE#MODEL`), and only on a pass writes `build/selfhost/model.json`, which `selfhost.sh` turns into the cell's model, embedder and profile; `make selfhost-bundle` carries the weights and the image. The runbook text (sb-selfhost(7)) states the hardware and latency. **In review:** the run itself, which needs the operator to fetch a GGUF instruct model and an embedding model and llama.cpp's server image (a download the user approves), then `make selfhost-model` and `make selfhost-smoke` on Docker Desktop.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
