---
title: "SB-2005: Mattermost entry point"
summary: "Sovereign sites rarely use Slack; they run their own chat, and the brain has no way in from it."
---

# SB-2005: Mattermost entry point

| Field | Value |
|---|---|
| Epic | [E20 Close the Loops](README.md) |
| Wave / slice | W19 / S19.3 |
| Stories | US-20.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-01, LAW-07, LAW-12 |
| Depends on | — |
| Status | Open |

## Why

Sovereign sites rarely use Slack; they run their own chat, and the brain has no way in from it.

## Acceptance criteria

- [ ] A Mattermost slash command reaches `sb-api` at `/chat/mattermost`, checked against the command's token by a vault reference
- [ ] The Mattermost user is matched by email to the directory, and the answer is the ask path under their access, posted only to them
- [ ] A user not in the directory, or deprovisioned, is refused

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Handler sharing the Slack path's rules
- [ ] Tests with a fake Mattermost
- [ ] Docs
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
