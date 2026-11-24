---
title: "SB-4006: Agents when something happens"
summary: "A joiner starts with nothing, and a transcript's decisions wait for someone to dig them out."
---

# SB-4006: Agents when something happens

| Field | Value |
|---|---|
| Epic | [E40 An Agent for Every Department](README.md) |
| Wave / slice | W39 / S39.2 |
| Stories | US-40.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-12, LAW-16 |
| Depends on | SB-3701 |
| Status | Done |

## Why

A joiner starts with nothing, and a transcript's decisions wait for someone to dig them out.

## Acceptance criteria

- [x] When SCIM adds someone to a space's group they get one welcome brief from the start-here lists of that space and those above it, each item checked as them (`welcome`, off-able)
- [x] A new transcript in a space drafts its decisions to the steward queue
- [x] Only these two triggers; each writes a trace and an audit row

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] SCIM hook
- [x] Ingest hook
- [x] Notify
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

SCIM joins queue one welcome brief per person and space (welcome, off-able); a new transcript drafts its decisions to the stewards; each writes a trace and an audit row

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
