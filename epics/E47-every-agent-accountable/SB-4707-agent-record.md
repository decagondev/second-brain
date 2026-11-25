---
title: "SB-4707: An agent's record"
summary: "An auditor asking how an agent behaved gets a dozen screens."
---

# SB-4707: An agent's record

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.2 |
| Stories | US-47.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-audit(1)`](../../manual/sb-audit.1.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4701, SB-4702, SB-4705 |
| Status | Done |

## Why

An auditor asking how an agent behaved gets a dozen screens.

## Acceptance criteria

- [x] `sb agents record NAME` and **Download record** write one signed JSON file and its Markdown
- [x] It holds the definition, owner, versions, eval scores, value, pauses, spend, access reviews and incidents
- [x] No document text or asker names; the signature verifies

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] CLI
- [x] API
- [x] Signing
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `ask.BuildRecord` gathers one agent's definition, owner, prompt SHA-256, eval scores, the versions it ran under, its value per quarter, pauses, spend, agent reviews and incident reports: counts, names of spaces and agents, never document text or askers. `SignRecord` signs its canonical JSON SHA-256 with the cell's anchor key (`audit.Signer.SignDocument`, kind `agent-record`), and `VerifyRecord` checks the hash, the signature and, when given, a pinned key. `GET /v1/admin/agents/NAME/record` (also `?format=md`) and `GET /v1/cell/agents/NAME/record` serve it; `sb agents record NAME` checks it before it writes NAME-record.json and NAME-record.md; the operators' Agents screen has **Download record** and its Markdown. TestRecordSignsAndCatchesTampering, TestProveItThroughARunningCell and TestAgentRecordAgainstARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
