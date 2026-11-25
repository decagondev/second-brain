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
| Status | Open |

## Why

An ambiguous ask gets the agent's best guess, and the person finds out only when the answer is wrong.

## Acceptance criteria

- [ ] A term with two glossary meanings, several matching spaces or a missing period gets one question with up to 4 choices and "just answer"
- [ ] Scheduled, triggered and shared runs never ask and state their assumption at the top
- [ ] A second question in one turn is refused; the choice is never kept in memory unless the person saves it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Ambiguity checks
- [ ] Question card in the web UI, Slack and Mattermost
- [ ] Assumption line
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
