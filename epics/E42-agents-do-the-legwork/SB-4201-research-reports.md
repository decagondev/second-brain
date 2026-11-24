---
title: "SB-4201: Research reports"
summary: "A question that needs many documents gets one shallow answer, and nobody has time to ask the ten follow-ups."
---

# SB-4201: Research reports

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.1 |
| Stories | US-42.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-14, LAW-26 |
| Depends on | SB-4105 |
| Status | Open |

## Why

A question that needs many documents gets one shallow answer, and nobody has time to ask the ten follow-ups.

## Acceptance criteria

- [ ] **Research** shows a plan of at most 10 sub-questions and runs only when the person presses **Run**
- [ ] Each sub-question is asked on the `ask` path as the person; the report cites every claim, and an uncited claim is dropped
- [ ] A run stops at `research_usd` or the person's space budget; the report is kept 30 days and every cite is checked again as the reader on opening (LAW-26)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent and prompt
- [ ] Job
- [ ] API
- [ ] UI
- [ ] Eval suite
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
