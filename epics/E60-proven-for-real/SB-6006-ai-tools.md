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
| Status | Open |

## Why

Nobody can see which AI tools reach the brain on a person's behalf, or turn one off.

## Acceptance criteria

- [ ] A token made for MCP names its client; My data lists each of the person's tools with its last use, its families and **Revoke**
- [ ] `GET /v1/admin/ai/clients` counts tokens, people and calls per client name, never per person
- [ ] sb-cell(5) `mcp_clients` may allow only named clients; a token for another can't be made

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Client names on tokens and traces
- [ ] My data list and the admin counts
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
