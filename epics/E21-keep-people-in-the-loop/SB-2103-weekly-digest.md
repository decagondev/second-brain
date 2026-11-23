---
title: "SB-2103: Weekly "what changed" digest"
summary: "'What changed' waits for someone to look."
---

# SB-2103: Weekly "what changed" digest

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.2 |
| Stories | US-21.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | SB-2101 |
| Status | Done |

## Why

"What changed" waits for someone to look.

## Acceptance criteria

- [x] Monday 08:00 in the tenant's time zone, each person who asked in the last 90 days gets the titles of up to ten changed documents in their spaces, each checked live as them, and a count of the rest
- [x] A document whose ACL dropped the person since it changed isn't named; a rebuild is never a change
- [x] Opt out per person; `digest: off` in the manifest turns it off (the manifest field is also its time zone)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Digest in the notify job
- [x] Manifest field
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: Done. The notify job's weekly part runs on the first run after Monday 08:00 in sb-tenant(5) `digest` (an IANA zone, `off`, default UTC; `time/tzdata` is embedded so any box can load it). Each active person with a trace in the last 90 days gets the titles of up to ten documents from `GET /v1/changes?since=7d` run as them at send time (`wire.Cell.ScopeAs` reads their groups then), and how many more; nothing changed sends nothing. A cell's first run marks the week done, so its first digest is next Monday's. Law tests: `TestLAW01_TheDigestNamesOnlyWhatThePersonCanOpenNow`, `TestLAW13_ARebuildIsNeverInTheDigest`, self-test `TestSelf_DigestLaw`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
