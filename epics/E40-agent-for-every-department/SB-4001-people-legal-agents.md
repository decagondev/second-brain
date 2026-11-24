---
title: "SB-4001: People and Legal agents"
summary: "People and Legal spaces have only `ask`."
---

# SB-4001: People and Legal agents

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

People and Legal spaces have only `ask`.

## Acceptance criteria

- [x] `policy-answer` quotes the policy that applies, with its version, review date and who to ask, and never answers about a named employee's case
- [x] `clause-finder` lists the clauses on a topic in the contracts the asker can open, one cited line each, grouped by contract
- [x] Each has an eval suite in the gate and is refused outside its template's spaces

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Prompts
- [x] Eval suites
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `policy-answer` (people) and `clause-finder` (legal): prompts in `internal/core/prompts/set/`, `ask.Departments`, the catalogue's defaults, `manifest.Agents`, the platform policy's `use_agent` list. `evals/suites/agent-policy-answer.yaml`, `agent-clause-finder.yaml` in the gate; law14departments covers every template.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
