---
title: "SB-4002: Sales, Support and Marketing agents"
summary: "Sales, Support and Marketing spaces have only `ask`."
---

# SB-4002: Sales, Support and Marketing agents

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

Sales, Support and Marketing spaces have only `ask`.

## Acceptance criteria

- [x] `account-brief` gives an account's history from what the asker can open, every line cited
- [x] `case-brief` gives a ticket's history and how similar tickets were solved, cited
- [x] `brand-check` checks a pasted draft against the brand guide, citing each rule, treating the draft as data and never rewriting it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Prompts
- [x] Eval suites
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `account-brief` (sales), `case-brief` (support) and `brand-check` (marketing), as SB-4001; `brand-check`'s prompt treats the draft in the question as material to check, never instructions. Eval suites in the gate.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
