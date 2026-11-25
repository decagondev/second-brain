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
| Status | Done |

## Why

A steward approves a skill or agent change without seeing how it answers their own questions.

## Acceptance criteria

- [x] A manifest diff that changes a skill or agent can carry up to 10 scenarios, run against the current and proposed versions side by side
- [x] Scenarios run as the steward only
- [x] A kept scenario joins the eval suite and runs in the gate

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Scenario format
- [x] Run scenarios in the diff review
- [x] Keep as eval case
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `internal/core/ask/scenarios.go`: a scenario is a question, an optional department agent, and at least one of `cites` (KIND:ID), `says` and `never`, one line of at most 300 characters each, at most 10 a run. `POST /v1/spaces/NAME/proposals/{id}/scenarios` runs each one twice as the steward who sends it, in the space, through `Agent.ProbeAs` (a trace, no conversation): once with the space's `agents` and skills as applied and once as the open proposal would make them (`withScenario` overrides only those two). The answer shows both sides with pass, why and the answer's lines and cites, for that steward only; the proposal keeps each question with pass or fail per side and who ran them, which `GET /v1/admin/proposals` shows the operator, and the run is audited. `keep` turns a scenario that passed on the proposal and names `cites` into an eval case run as that steward, counted toward the 50. Tested in TestScenariosSourcesAndStartersThroughARunningCell and TestScenariosAreCheckedAndJudged. Scenarios are over the API; the steward screen has no editor for them yet.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
