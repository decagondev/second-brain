---
title: "SB-1102: Department agents"
summary: "A few departments have one job that a plain ask does badly."
---

# SB-1102: Department agents

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.1 Catalogue |
| Stories | US-11.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | [LAW-09](../../manual/sb-laws.7.md#law-09), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-1101](SB-1101-template-catalogue-as-data-and-sb-space-init.md) |
| Status | Done |

## Why

A few departments have one job that a plain ask does badly.

## Acceptance criteria

- [x] Read-only agents pinned in the bundle: `incident-summary` (Engineering), `spec-digest` (Product), `close-checklist` (Finance), `runbook-answer` (IT)
- [x] Each agent runs only in its template's spaces and cites every claim
- [x] Each has an eval suite

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent definitions
- [x] Space binding
- [x] Eval suites
- [x] Write or extend the law tests: LAW-09, LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S11.1b): sb-agents(7) gained "Department agents" and four rows; sb-ask(3) the `agent` field, `agent` and `dropped` in the response and the agent `EINVAL`; sb-space(5) rule 11; sb-tenant(5), sb-eval(5) and sb-evals(7) changed with them. The four prompts are bundled (`internal/core/prompts/set/`, so the release pins their hashes), and `ask.Departments` binds each to its template with no tools. The ask agent runs one only when the manifest enables it, the ask is in a space of its template that lists it, its kill switch is clear and the kernel permits `use_agent` (a new base permit for the four, enabled and not consequential); otherwise `EINVAL` before retrieval. It composes with its own prompt and drops every uncited claim, counting them. The manifest knows the four agents and refuses another department's agent in a space. The local eval cell takes a case's `agent`, a world's `agents` and a space's `template` and `agents`, and an `error` expectation; four suites, `agent-*.yaml`, pass and are in the gate. Law tests: `TestLAW14_DepartmentAgentsCiteEveryClaimInTheirOwnSpaces` (the real ask path with a model that spoofs and leaves claims uncited; an agent that keeps uncited claims, and one that ignores the template, are caught) and `TestLAW09_DepartmentAgentsCantAct` (a department agent with a tool is caught).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
