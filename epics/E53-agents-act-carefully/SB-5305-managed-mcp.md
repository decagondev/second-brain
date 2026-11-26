---
title: "SB-5305: Managed MCP access"
summary: "MCP is on or off for the whole cell, with every tool for everyone."
---

# SB-5305: Managed MCP access

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.2 |
| Stories | US-53.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-3801, SB-5001 |
| Status | Done |

## Why

MCP is on or off for the whole cell, with every tool for everyone.

## Acceptance criteria

- [x] sb-cell(5) `mcp.groups` names which IdP groups may connect MCP clients and which tools each sees
- [x] A member outside the groups is refused and a tool not granted isn't listed
- [x] The command centre shows use by client (web, chat, MCP, API token), matching the traces

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Group and tool lists
- [x] Refusal and listing
- [x] Use by client
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. sb-cell(5) `mcp_groups` (group → tool families of `search`, `ask` (with `passage`), `agents`, `research`, `skills`), checked at start (`Config.checkMCPGroups`, needs `mcp: true`); `api.MCP.Families` (from `Cell.mcpFamilies`) refuses a person in no listed group on every method, and `tools/list`, `tools/call` and the prompts offer only the granted families. Every ask records its client in `Trace.Client` (`tenancy.WithClient`: `token` for an API token, `chat` from the Slack and Mattermost apps, `mcp` from the endpoint, else `web`), and the AI command centre counts `clients`. The setting is beside `mcp` rather than replacing it, so a cell's existing `mcp: true` still means the same. Tested in TestManagedMCPAccessThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
