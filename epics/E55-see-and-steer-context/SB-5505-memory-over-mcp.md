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
| Status | Done |

## Why

Memory kept in the brain isn't available in the other tools a person uses.

## Acceptance criteria

- [x] The MCP server offers the person's memory lines as a read-only resource and a `memory_propose` tool
- [x] A proposed line goes to the person's inbox and counts only after they confirm it in the web UI
- [x] Groups without MCP access don't see either, and another person's memory is never offered

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] MCP resource and tool
- [x] Inbox kind `memory`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `Memory.Proposed` keeps lines an MCP client proposes (`Agent.ProposeMemory`, the same checks as Remember, at most 10); the MCP endpoint answers `resources/list` and `resources/read` with `memory://me`, the caller's own lines, and lists `memory_propose`, both in a new `memory` family of sb-cell(5) `mcp_groups`. A proposal shows in the inbox (kind `memory`) and on About me with Confirm and Dismiss (`Agent.DecideProposal`, `POST|DELETE /v1/me/memory/proposals/{id}`), which an API token can't call; until then it is never sent with an ask. Tested in TestAskWhyThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
