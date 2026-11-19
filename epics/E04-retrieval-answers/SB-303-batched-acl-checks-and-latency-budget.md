---
title: "SB-303: Batched ACL checks and latency budget"
summary: "Soundness first, then make it fast without caching allows across requests."
---

# SB-303: Batched ACL checks and latency budget

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.1 Retrieval |
| Stories | US-04.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-302](../E04-retrieval-answers/SB-302-live-authorisation-of-every-chunk-before.md) |
| Status | In review |

## Why

Soundness first, then make it fast without caching allows across requests.

## Acceptance criteria

- [ ] p95 ask latency 6 seconds or less in staging
- [x] No allow result is reused across requests (ADR-0002)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Batch API use per connector
- [x] Parallel fan-out with deadline
- [ ] Latency dashboards (SB-607)
- [x] Write or extend the law tests: LAW-01
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.1): batches of up to 200 documents per connector, all connectors in parallel, each with the 800 ms deadline. Tests cover batch splitting, the four ways of not answering (no connector, error, late, short answer), and two requests making two live calls (no allow caching), all under `-race`.

**Left:** p95 ask latency in staging (needs a staging cell) and the latency dashboards (SB-607).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
