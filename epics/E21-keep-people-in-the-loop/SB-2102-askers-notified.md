---
title: "SB-2102: Askers hear about replies"
summary: "An asker finds a steward's reply only by opening their questions."
---

# SB-2102: Askers hear about replies

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.1 |
| Stories | US-21.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-2101 |
| Status | Open |

## Why

An asker finds a steward's reply only by opening their questions.

## Acceptance criteria

- [ ] The notify job tells an asker "Your question has a reply" with a link to their questions, when a steward replies or publishes a note
- [ ] The message carries no reply text and no cite; the link opens the reply under the asker's access
- [ ] Each reply is told once

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Reply events in the job
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
