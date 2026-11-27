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
| Status | Open |

## Why

A shared mailbox such as support@ can't start an agent.

## Acceptance criteria

- [ ] sb-space(5) `mailbox` names one shared mailbox and `on: [mail-received]` starts the space's agent on a new message
- [ ] The agent drafts a reply note for a steward to edit and send themselves; attachments are read like files; the mail is data
- [ ] A mailbox not named in the manifest, and a person's own mailbox, are never read

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Shared mailbox reader over Graph
- [ ] `mail-received` event
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
