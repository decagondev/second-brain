---
title: "SB-5502: Keep or drop for this conversation"
summary: "Steering an answer means rewording the question and hoping."
---

# SB-5502: Keep or drop for this conversation

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.1 |
| Stories | US-55.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01 |
| Depends on | SB-5501, SB-3004, SB-2405 |
| Status | Done |

## Why

Steering an answer means rewording the question and hoping.

## Acceptance criteria

- [x] From the panel, **Keep** adds a document to every later ask in the conversation and **Drop** leaves it out
- [x] Both live with the conversation in the browser and go into its download; nothing is stored on the server
- [x] A kept document is checked as the person on each ask and stops being used if they lose access

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Web: keep and drop
- [x] Request carries kept and dropped documents
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. The ask takes `keep` and `drop` (KIND:ID, at most 20 each, `steerOfRequest`); `Agent.steered` drops the dropped documents and puts the kept ones' first passages first, each through the live check, retirement, hold and exclusions as the asker. The web keeps both per conversation in the browser (`App` `steers`), sends them with each ask and adds them to the conversation's download; the server keeps only the trace's ids.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
