---
title: "SB-4404: Signed events out"
summary: "Other systems can't react to what happens in the brain without polling it."
---

# SB-4404: Signed events out

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.2 |
| Stories | US-44.2 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-2306 |
| Status | Done |

## Why

Other systems can't react to what happens in the brain without polling it.

## Acceptance criteria

- [x] `events: { url, kinds }` is off by default; chosen kinds post the kind, space, id and time only
- [x] Each body is signed with HMAC-SHA256 from a vault key reference, retried with backoff and audited
- [x] A receiver fetches more with its own read-only token, checked as that token's person

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest
- [x] Delivery
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. sb-tenant(5) `events: { url, kinds }` (https only, known kinds once each), off by default; key at `vault://tenants/TENANT/events`. internal/core/events: `{kind, space, id, at}` signed with HMAC-SHA256 over the timestamp and body (`X-SB-Timestamp`, `X-SB-Signature`), two retries with backoff in a run, each event once per space (state/events/sent.json), every attempt audited (`access`, `event_sent` or `event_failed`). The notify job runs it; sources are decision passages and policies new in 7 days, goals at risk or off track once per version, overdue requests and action items past due. The receiver reads more with its own read-only token. TestEventsAreSignedRetriedAndSentOnce, TestEventsAndModelPin, TestEventsTiersAndGuardrailsThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
