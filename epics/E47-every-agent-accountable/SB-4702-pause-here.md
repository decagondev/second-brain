---
title: "SB-4702: Pause from the message"
summary: "Stopping a misbehaving agent in one space means finding an admin."
---

# SB-4702: Pause from the message

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.1 |
| Stories | US-47.1 |
| Manual | [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-10, LAW-11 |
| Depends on | SB-4402 |
| Status | Done |

## Why

Stopping a misbehaving agent in one space means finding an admin.

## Acceptance criteria

- [x] Every agent message and draft carries **Pause here** for the space's stewards and the agent's owner
- [x] The agent stops in that space at once, the reason goes to the owner, and it's audited
- [x] A steward or the owner resumes it; waiting drafts are kept

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Kill switch scope
- [x] Chat and UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. A pause is the kill switch `agent:NAME@SPACE`: `kill.Checker.Check` refuses an `Op` with that agent and space, and `Paused` answers paused while the list can't be read. Drafts, routine runs, redrafts and named-agent asks check it (`EKILLED`); people's own asks with `ask` go on. `POST /v1/spaces/NAME/agents/AGENT/pause|resume` takes a reason, is allowed to the space's stewards and the agent's owner only, audits `kind: kill`, refreshes the checker at once and tells the owner. Agents speak in a space only through drafts in its steward queue (routine results are kept for the person alone), so Pause here sits on every agent draft there and on an owner's value rows; the space's agents list carries `paused`. LAW-10 now also sets and checks `agent:NAME@SPACE` scopes on its fake clock. TestAgentPausedInASpace, TestLAW10_KillWins, TestEveryAgentAccountableThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
