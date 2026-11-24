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
| Status | Open |

## Why

People using other AI tools can ask the brain, but not its department agents.

## Acceptance criteria

- [ ] The MCP server lists as tools the department agents the token's person may use, plus `research_plan` and `research_report`
- [ ] A tool call runs exactly as `ask` does for that agent, as the person; nothing writes
- [ ] A killed agent disappears from the list at once

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] MCP
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
