---
title: "SB-5304: Draft the next action"
summary: "An answer that recommends a ticket or a message leaves the person to write it from scratch."
---

# SB-5304: Draft the next action

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.2 |
| Stories | US-53.2 |
| Manual | [`sb-actions(3)`](../../manual/sb-actions.3.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-04, LAW-16 |
| Depends on | SB-4301, SB-2105 |
| Status | Open |

## Why

An answer that recommends a ticket or a message leaves the person to write it from scratch.

## Acceptance criteria

- [ ] **Draft it** prepares a Linear ticket or a Slack or Mattermost message from an answer, brief or review
- [ ] The person edits it and presses **Send**, which goes out with their own token through the tool bus; agents never send
- [ ] A cite they can't open isn't linked, and the sent item is audited

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Linear and chat drafts
- [ ] Send through the tool bus
- [ ] Tests
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: waiting on a decision. Sending "with the person's own token" needs per-person sign-in to Linear, Slack and Mattermost and a store for those tokens, which this wave's rules leave out. The choices put to the user: send through the tool bus with the tenant's credentials (a Linear comment through the gated ticket-update workflow, a message by the bot marked as sent by the person, nothing out before they press Send); the person posts it from the tool itself, the brain only preparing it; or leave SB-5304 out of W52.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
