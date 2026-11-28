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
| Status | Done |

## Why

A Jira Service Management queue can't start an agent.

## Acceptance criteria

- [x] sb-space(5) `service_queue` names one queue and `on: [request-received]` starts the space's agent on a new request
- [x] The agent drafts a reply note for a service agent to edit and send themselves; the request is data
- [x] A queue not named in the manifest is never read, and nothing is sent

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Queue reader
- [x] `request-received` event
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. sb-space(5) `service_queue` and `on: [request-received]`; `Cell.serviceEvents` (with the mailboxes, in the notify job) reads the queue's requests created since `state/requests/SPACE`, the first read from now, and drafts each as kind `request` with the request as a file read for that ask alone; nothing is sent. Tested in TestAtlassianThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
