---
title: "SB-3801: Use the brain from other AI tools"
summary: "Other AI tools can't use the brain, so people paste between them."
---

# SB-3801: Use the brain from other AI tools

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.1 |
| Stories | US-38.1 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-01, LAW-11, LAW-12 |
| Depends on | SB-2306 |
| Status | Done |

## Why

Other AI tools can't use the brain, so people paste between them.

## Acceptance criteria

- [x] `POST /v1/mcp` speaks MCP with read-only search, ask and passage tools, taking the person's sbt_ token
- [x] Every result is checked live as them, at the same rate limit; no tool writes
- [x] Off unless sb-cell(5) `mcp: true`; a revoked token is refused

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] MCP handler
- [x] Wiring
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `internal/api/mcp.go` serves `POST /v1/mcp` (MCP 2025-06-18 over streamable HTTP, one JSON response per request): `initialize`, `ping`, `tools/list`, `tools/call` with read-only `search`, `ask` and `passage` tools that call the same agent methods as their paths, as the caller. sb-cell(5) `mcp: true` mounts it (`wire.Config.MCP`); `session.TokenRoute` lets an sbt_ token reach it. TestWorkWherePeopleAreThroughARunningCell (token use audited, a revoked token refused at once), TestMCPOffByDefault, the TokenRoute unit test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
