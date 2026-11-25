---
title: "SB-4904: Scenarios before you approve"
summary: "A steward approves a skill or agent change without seeing how it answers their own questions."
---

# SB-4904: Scenarios before you approve

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.2 |
| Stories | US-49.2 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-06 |
| Depends on | SB-4601, SB-4401 |
| Status | Open |

## Why

A steward approves a skill or agent change without seeing how it answers their own questions.

## Acceptance criteria

- [ ] A manifest diff that changes a skill or agent can carry up to 10 scenarios, run against the current and proposed versions side by side
- [ ] Scenarios run as the steward only
- [ ] A kept scenario joins the eval suite and runs in the gate

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Scenario format
- [ ] Run scenarios in the diff review
- [ ] Keep as eval case
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
