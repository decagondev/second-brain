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
| Status | Done |

## Why

MCP clients that look for skills through `skills/list` find none.

## Acceptance criteria

- [x] The MCP server answers `skills/list` and `skills/get` beside `prompts/list` and `prompts/get`
- [x] Only skills of agents the token's person can use, in groups allowed MCP
- [x] The text is the bundle's signed skill

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `skills/list` and `skills/get` in `api.MCP`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. The MCP server answers `skills/list` and `skills/get` beside `prompts/list` and `prompts/get`, with the same skills (`UsableSkills`: the spaces the token's person is a member of now) and only for groups granted the `skills` family; each is `{ name, description, instructions }` from the release bundle's skill text, and `initialize` declares `skills`. Tested in TestShapeTheWorkThroughARunningCell and TestManagedMCPAccessThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
