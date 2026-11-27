---
title: "SB-5807: Skills over MCP"
summary: "MCP clients that look for skills through `skills/list` find none."
---

# SB-5807: Skills over MCP

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.2 |
| Stories | US-58.2 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-09 |
| Depends on | SB-4601, SB-4306, SB-5305 |
| Status | Open |

## Why

MCP clients that look for skills through `skills/list` find none.

## Acceptance criteria

- [ ] The MCP server answers `skills/list` and `skills/get` beside `prompts/list` and `prompts/get`
- [ ] Only skills of agents the token's person can use, in groups allowed MCP
- [ ] The text is the bundle's signed skill

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `skills/list` and `skills/get` in `api.MCP`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
