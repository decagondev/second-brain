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
| Status | Open |

## Why

Work events, incident briefs and reviews start only on Linear.

## Acceptance criteria

- [ ] `on: [ticket-filed]` starts on a filed Jira issue
- [ ] An incident label starts a brief and closing the issue starts a review, as on Linear
- [ ] Events come from the next sync or a Jira webhook verified with its secret; an unsigned webhook is refused

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Jira events in the notify job
- [ ] Webhook verification
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
