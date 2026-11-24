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
| Status | Open |

## Why

Other AI tools can't use the brain, so people paste between them.

## Acceptance criteria

- [ ] `POST /v1/mcp` speaks MCP with read-only search, ask and passage tools, taking the person's sbt_ token
- [ ] Every result is checked live as them, at the same rate limit; no tool writes
- [ ] Off unless sb-cell(5) `mcp: true`; a revoked token is refused

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] MCP handler
- [ ] Wiring
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
