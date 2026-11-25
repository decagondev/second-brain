---
title: "SB-4901: Ask before guessing"
summary: "An ambiguous ask gets the agent's best guess, and the person finds out only when the answer is wrong."
---

# SB-4901: Ask before guessing

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.1 |
| Stories | US-49.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-09, LAW-26 |
| Depends on | SB-3902, SB-4007 |
| Status | Done |

## Why

An ambiguous ask gets the agent's best guess, and the person finds out only when the answer is wrong.

## Acceptance criteria

- [x] A term with two glossary meanings, several matching spaces or a missing period gets one question with up to 4 choices and "just answer"
- [x] Scheduled, triggered and shared runs never ask and state their assumption at the top
- [x] A second question in one turn is refused; the choice is never kept in memory unless the person saves it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ambiguity checks
- [x] Question card in the web UI (Slack and Mattermost state the assumption instead)
- [x] Assumption line
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `internal/core/ask/clarify.go`: before it searches, an ask looks for a glossary term two spaces in scope define differently, a company-wide question using terms of several departments, or an amount or trend question naming no period, in that order, on a conversation's first question only. With `clarify: true` (the web UI) the response is `clarify` (kind, question, up to 4 choices and `just answer`) and nothing is composed or kept; `choice` answers it and goes into the question as `The asker means: …`, which the conversation keeps (never memory). A choice the question wouldn't be offered is `EINVAL`, so an ask is asked about once. Every other caller (chat apps, MCP, routines, research steps, scheduled and triggered agents, rhythm steps) takes the first choice and says so in `assumed`, which `ask.Warnings` puts first in Slack and Mattermost and agent drafts start with. Web: `Clarify.tsx`. Tested in TestAskBeforeGuessing and clarify.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
