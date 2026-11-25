---
title: "SB-4402: Agent owners and changes"
summary: "Nobody is answerable for an agent, and a prompt change ships without anyone seeing it."
---

# SB-4402: Agent owners and changes

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.1 |
| Stories | US-44.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-release(1)`](../../manual/sb-release.1.md) |
| Laws | LAW-06, LAW-16 |
| Depends on | SB-4004 |
| Status | Done |

## Why

Nobody is answerable for an agent, and a prompt change ships without anyone seeing it.

## Acceptance criteria

- [x] Every agent in the bundle names an `owner`; the gate refuses one without; the registry shows it and flags an inactive owner
- [x] `sb release notes` lists each agent's prompt diff and eval change
- [x] An agent not run for 60 days in a space is proposed off as a manifest diff, never turned off

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Bundle
- [x] Registry
- [x] Release notes
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. internal/core/ask/agents.json is every agent's definition in the bundle, with its owner (an email or group:NAME). The registry shows owner and owner_inactive (wire: one active user for an email, any active member for a group); a space's agents list proposes off a department agent with no run there in 60 days, only when traces are kept that long. `sb release notes --from DIR --to DIR` lists each agent's prompt diff and eval change; release-build copies the definitions, prompts and results to build/release/agents/. TestAgentOwnersAndUnused.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
