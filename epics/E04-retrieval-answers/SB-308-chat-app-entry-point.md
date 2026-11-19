---
title: "SB-308: Chat-app entry point"
summary: "Meet people where they already ask questions."
---

# SB-308: Chat-app entry point

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.4 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md) |
| Status | In review |

## Why

Meet people where they already ask questions.

## Acceptance criteria

- [ ] Ask from the company chat app; replies are visible only to the asker unless they share (built and tested against fakes; a run in a real Slack workspace is left)
- [x] Same API and laws as web

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Bot adapter
- [x] Ephemeral replies
- [x] Identity mapping chat user to IdP user
- [x] Write or extend the law tests: LAW-01
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.5): `internal/adapters/chatapp/slackapp` serves `/ask` (`POST /chat/slack/commands`) and the Share button (`POST /chat/slack/interactions`). It verifies Slack's v0 signature within 5 minutes and the tenant's `team_id`, and accepts only `https://hooks.slack.com/` response URLs. It maps the Slack user's `users.info` email to exactly one active directory user (new `ports.UserFinder`, implemented by the memory directory), and issues their scope through the cell's one issuer. It acknowledges at once and posts the answer ephemerally through `response_url`, with citations as links and uncited claims labelled, all Slack markup escaped. Share posts citation links only, never claim text, and only the asker can share (the trace is read with their own scope). `TestLAW01_ChatAnswersCiteOnlyWhatTheAskerCanOpen` went red first, then green through the real app, mapping and agent, and a mutation that answers as the wrong person is caught.

**Left:** a run in a real Slack workspace (app manifest with the `commands` scope, slash command and interactivity URLs).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
