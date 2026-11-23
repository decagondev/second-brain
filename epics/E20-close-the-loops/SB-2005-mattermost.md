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
| Status | Done |

## Why

Sovereign sites rarely use Slack; they run their own chat, and the brain has no way in from it.

## Acceptance criteria

- [x] A Mattermost slash command reaches `sb-api` at `/chat/mattermost`, checked against the command's token by a vault reference
- [x] The Mattermost user is matched by email to the directory, and the answer is the ask path under their access, posted only to them
- [x] A user not in the directory, or deprovisioned, is refused

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Handler sharing the Slack path's rules
- [x] Tests with a fake Mattermost
- [x] Docs
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built 2026-11-23. `internal/adapters/chatapp/mmapp`: a Mattermost custom slash command at `POST /chat/mattermost`, verified by `command_token` from `vault://tenants/TENANT/mattermost` (a new allowed vault path) and the tenant's `team_id`. The person is mapped with the Slack app's own rules (`slackapp.App.Scope` and `InSpace`, now exported): the Mattermost user's email from `GET /api/v4/users/ID` with `bot_token`, to exactly one active directory user; space channels ask in their space. The answer is ephemeral, through the server's own `/hooks/commands/` response URL, in escaped Markdown. sb-cell(5) `mattermost.base` and `mattermost.team_id` turn it on. Tests: `mmapp_test.go` with a fake Mattermost (privacy, refusals of unknown, deprovisioned, deleted and bot users, token, team and response URL checks), `tests/cell/mattermost_test.go` through a running cell. Laws: `TestLAW01_MattermostAnswersCiteOnlyWhatTheAskerCanOpen`, `TestLAW12_MattermostRefusesTheDeprovisioned` (self-test catches a mapping that answers the deprovisioned); LAW-07 masking is the ask path's. Manual: sb-ask(3), sb-cell(5), sb-tenant(5); runbook section.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
