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
| Status | Open |

## Why

"What changed" waits for someone to look.

## Acceptance criteria

- [ ] Monday 08:00 in the tenant's time zone, each person who asked in the last 90 days gets the titles of up to ten changed documents in their spaces, each checked live as them, and a count of the rest
- [ ] A document whose ACL dropped the person since it changed isn't named; a rebuild is never a change
- [ ] Opt out per person; `digest: false` in the manifest turns it off

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Digest in the notify job
- [ ] Manifest field
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
