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
| Status | Open |

## Why

Nobody is answerable for an agent, and a prompt change ships without anyone seeing it.

## Acceptance criteria

- [ ] Every agent in the bundle names an `owner`; the gate refuses one without; the registry shows it and flags an inactive owner
- [ ] `sb release notes` lists each agent's prompt diff and eval change
- [ ] An agent not run for 60 days in a space is proposed off as a manifest diff, never turned off

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Bundle
- [ ] Registry
- [ ] Release notes
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
