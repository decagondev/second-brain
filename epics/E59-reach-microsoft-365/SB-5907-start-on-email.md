---
title: "SB-5907: Start on a new email"
summary: "A shared mailbox such as support@ can't start an agent."
---

# SB-5907: Start on a new email

| Field | Value |
|---|---|
| Epic | [E59 Reach Microsoft 365](README.md) |
| Wave / slice | W58 / S58.2 |
| Stories | US-59.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-09, LAW-16 |
| Depends on | SB-5301, SB-5304, SB-3802 |
| Status | Done |

## Why

A shared mailbox such as support@ can't start an agent.

## Acceptance criteria

- [x] sb-space(5) `mailbox` names one shared mailbox and `on: [mail-received]` starts the space's agent on a new message
- [x] The agent drafts a reply note for a steward to edit and send themselves; attachments are read like files; the mail is data
- [x] A mailbox not named in the manifest, and a person's own mailbox, are never read

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Shared mailbox reader over Graph
- [x] `mail-received` event
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. sb-space(5) `mailbox` and `on: [mail-received]` (an error without the mailbox); `msgraph.Mailbox.New` reads the shared mailbox's delta from `state/mail/SPACE`, starting from now; `Cell.mailEvents` in the notify job drafts each new message as kind `mail` ("Mail: SUBJECT") with `ask.MailQuestion` and the message and its attachments' text as a file read for that ask alone (data, LAW-09); nothing is sent. Building it found BUG-0031 (work-event drafts were refused by the draft kinds check), fixed. Tested in TestSharedMailbox, TestWorkAndMailDraftsArePut and TestMicrosoft365ThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
