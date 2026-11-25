---
title: "SB-4306: Agents over MCP"
summary: "People using other AI tools can ask the brain, but not its department agents."
---

# SB-4306: Agents over MCP

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.2 |
| Stories | US-43.2 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-10 |
| Depends on | SB-3801 |
| Status | Done |

## Why

People using other AI tools can ask the brain, but not its department agents.

## Acceptance criteria

- [x] The MCP server lists as tools the department agents the token's person may use, plus `research_plan` and `research_report`
- [x] A tool call runs exactly as `ask` does for that agent, as the person; nothing writes
- [x] A killed agent disappears from the list at once

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] MCP
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Agent.UsableAgents (every pre-retrieval check of an ask with the agent, as the person in each of their spaces), api.MCP Agents/ScopeIn/Legwork, tools agent_NAME with a space enum, research_plan and research_report, the list made at each tools/list. A killed agent is gone at once (TestContextForTheWorkThroughARunningCell).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
