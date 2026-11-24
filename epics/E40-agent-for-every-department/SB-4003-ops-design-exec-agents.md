---
title: "SB-4003: Operations, Design and Executive agents"
summary: "Operations, Design and Executive spaces have only `ask`."
---

# SB-4003: Operations, Design and Executive agents

| Field | Value |
|---|---|
| Epic | [E40 An Agent for Every Department](README.md) |
| Wave / slice | W39 / S39.1 |
| Stories | US-40.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-departments(7)`](../../manual/sb-departments.7.md) |
| Laws | LAW-01, LAW-09, LAW-14 |
| Depends on | SB-1102 |
| Status | Done |

## Why

Operations, Design and Executive spaces have only `ask`.

## Acceptance criteria

- [x] `sop-steps` gives a procedure's steps in order, each cited, with the version
- [x] `design-rationale` says why a design is as it is, from specs, decisions and reviews
- [x] `exec-brief` gives what the leadership team decided this month and what's open, only in a restricted space made from `executive`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Prompts
- [x] Eval suites
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `sop-steps` (operations), `design-rationale` (design) and `exec-brief` (executive), as SB-4001; `Department.Restricted` runs `exec-brief` only in a restricted space, and law14departments draws restricted and internal spaces to check it. Eval suites in the gate.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
