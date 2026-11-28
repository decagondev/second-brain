---
title: "SB-6205: Jira as work events"
summary: "Work events, incident briefs and reviews start only on Linear."
---

# SB-6205: Jira as work events

| Field | Value |
|---|---|
| Epic | [E62 Reach Confluence and Jira](README.md) |
| Wave / slice | W61 / S61.2 |
| Stories | US-62.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-09, LAW-16 |
| Depends on | SB-5301, SB-4204, SB-4604, SB-6204 |
| Status | Done |

## Why

Work events, incident briefs and reviews start only on Linear.

## Acceptance criteria

- [x] `on: [ticket-filed]` starts on a filed Jira issue
- [x] An incident label starts a brief and closing the issue starts a review, as on Linear
- [x] Events come from the next sync or a Jira webhook verified with its secret; an unsigned webhook is refused

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Jira events in the notify job
- [x] Webhook verification
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. Work events, incident briefs and reviews are vendor-neutral on the tickets kind and its `Labels:` and `Status:` lines, so Jira issues start them as Linear's do; `POST /hooks/jira`, verified with `webhook_secret` (`X-Hub-Signature: sha256=`), starts a tickets sync at once. Tested in TestAtlassianThroughARunningCell (an unsigned delivery is refused, a signed one accepted).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
