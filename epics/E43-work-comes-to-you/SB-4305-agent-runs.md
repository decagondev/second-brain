---
title: "SB-4305: What agents did as me"
summary: "Scheduled briefs, triggers and routines run as people, who can't see what ran in their name."
---

# SB-4305: What agents did as me

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.2 |
| Stories | US-43.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-01, LAW-11 |
| Depends on | SB-4005 |
| Status | Open |

## Why

Scheduled briefs, triggers and routines run as people, who can't see what ran in their name.

## Acceptance criteria

- [ ] `GET /v1/me/agent-runs` and **Agent runs** list every run made as me in the last 90 days, from traces and the audit log
- [ ] Each shows when, which agent and space, what it read (titles I can open now, the rest counted) and what it drafted
- [ ] No new record is kept

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
