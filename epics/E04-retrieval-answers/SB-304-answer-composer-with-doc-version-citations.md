---
title: "SB-304: Answer composer with `doc@version` citations and `uncited` in code"
summary: "Citations are structure, not prose; the model proposes, code disposes."
---

# SB-304: Answer composer with `doc@version` citations and `uncited` in code

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.1, US-04.2 |
| Manual | [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-302](../E04-retrieval-answers/SB-302-live-authorisation-of-every-chunk-before.md) |
| Status | Done |

## Why

Citations are structure, not prose; the model proposes, code disposes.

## Acceptance criteria

- [x] Every claim span maps to an allowed chunk citation or is labelled `uncited` by code
- [x] Citations that don't match an allowed chunk are stripped and logged

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Claim-span schema
- [x] Citation validator
- [x] `uncited` labeller
- [x] LAW-14 property test
- [x] Write or extend the law tests: LAW-01, LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.2, with SB-305): `internal/core/answer`. `NewContext` labels allowed chunks C1, C2, ... and `Data` renders them as data blocks. `Compose` parses the model's JSON, maps labels to chunks, strips any label outside the context (returned as `Spoofs`, ids only, for the ask agent to log as `citation_spoof` in SB-306), and marks a claim `uncited` when nothing survives. Output that isn't the expected JSON becomes one uncited claim. `TestLAW14_EveryClaimCitedOrLabelled` is green on the real composer, and a mutation that trusts the model's labels is caught.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
