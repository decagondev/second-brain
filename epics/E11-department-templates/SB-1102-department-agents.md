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
| Status | Open |

## Why

A few departments have one job that a plain ask does badly.

## Acceptance criteria

- [ ] Read-only agents pinned in the bundle: `incident-summary` (Engineering), `spec-digest` (Product), `close-checklist` (Finance), `runbook-answer` (IT)
- [ ] Each agent runs only in its template's spaces and cites every claim
- [ ] Each has an eval suite

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent definitions
- [ ] Space binding
- [ ] Eval suites
- [ ] Write or extend the law tests: LAW-09, LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
