---
title: "SB-6006: The AI tools that reach the brain"
summary: "Nobody can see which AI tools reach the brain on a person's behalf, or turn one off."
---

# SB-6006: The AI tools that reach the brain

| Field | Value |
|---|---|
| Epic | [E60 Proven for Real](README.md) |
| Wave / slice | W59 / S59.2 |
| Stories | US-60.2 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-01, LAW-11 |
| Depends on | SB-3801, SB-5305, SB-2306 |
| Status | Done |

## Why

Nobody can see which AI tools reach the brain on a person's behalf, or turn one off.

## Acceptance criteria

- [x] A token made for MCP names its client; My data lists each of the person's tools with its last use, its families and **Revoke**
- [x] `GET /v1/admin/ai/clients` counts tokens, people and calls per client name, never per person
- [x] sb-cell(5) `mcp_clients` may allow only named clients; a token for another can't be made

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Client names on tokens and traces
- [x] My data list and the admin counts
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. A token may name its AI tool (`client`); sb-cell(5) `mcp_clients` makes it required and limits MCP to listed tools' tokens; the auth middleware puts the tool in the request (`tenancy.WithTool`) and traces keep it (`Trace.Tool`). `GET /v1/me/ai-tools` and My data's **AI tools that reach the brain for me** list the person's tools with last use, families and **Revoke**; `GET /v1/admin/ai/clients` counts tokens, people and this month's asks per tool, never who. Tested in TestAIToolsThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
