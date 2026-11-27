---
title: "SB-6206: Start on a service request"
summary: "A Jira Service Management queue can't start an agent."
---

# SB-6206: Start on a service request

| Field | Value |
|---|---|
| Epic | [E62 Reach Confluence and Jira](README.md) |
| Wave / slice | W61 / S61.2 |
| Stories | US-62.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-09, LAW-16 |
| Depends on | SB-5907, SB-5304, SB-6204 |
| Status | Open |

## Why

A Jira Service Management queue can't start an agent.

## Acceptance criteria

- [ ] sb-space(5) `service_queue` names one queue and `on: [request-received]` starts the space's agent on a new request
- [ ] The agent drafts a reply note for a service agent to edit and send themselves; the request is data
- [ ] A queue not named in the manifest is never read, and nothing is sent

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Queue reader
- [ ] `request-received` event
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
