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
| Status | Open |

## Why

MCP is on or off for the whole cell, with every tool for everyone.

## Acceptance criteria

- [ ] sb-cell(5) `mcp.groups` names which IdP groups may connect MCP clients and which tools each sees
- [ ] A member outside the groups is refused and a tool not granted isn't listed
- [ ] The command centre shows use by client (web, chat, MCP, API token), matching the traces

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Group and tool lists
- [ ] Refusal and listing
- [ ] Use by client
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
