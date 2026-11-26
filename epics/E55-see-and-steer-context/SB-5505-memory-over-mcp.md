---
title: "SB-5505: Memory over MCP"
summary: "Memory kept in the brain isn't available in the other tools a person uses."
---

# SB-5505: Memory over MCP

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.2 |
| Stories | US-55.2 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-09, LAW-25 |
| Depends on | SB-3801, SB-4303, SB-5305 |
| Status | Open |

## Why

Memory kept in the brain isn't available in the other tools a person uses.

## Acceptance criteria

- [ ] The MCP server offers the person's memory lines as a read-only resource and a `memory_propose` tool
- [ ] A proposed line goes to the person's inbox and counts only after they confirm it in the web UI
- [ ] Groups without MCP access don't see either, and another person's memory is never offered

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] MCP resource and tool
- [ ] Inbox kind `memory`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
