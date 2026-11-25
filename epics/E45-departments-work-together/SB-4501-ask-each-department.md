---
title: "SB-4501: Ask each department"
summary: "A question that spans departments gets one department's view, and the asker has to ask each one in turn."
---

# SB-4501: Ask each department

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.1 |
| Stories | US-45.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-24, LAW-27 |
| Depends on | SB-3903, SB-4007 |
| Status | Done |

## Why

A question that spans departments gets one department's view, and the asker has to ask each one in turn.

## Acceptance criteria

- [x] A branch ask consults at most 6 child spaces the asker is a member of, each agent answering as the asker from its own space
- [x] The answer has a section per department and says which weren't consulted and why
- [x] A consulted agent never consults or calls a tool; LAW-27 is a law test and goes green

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Consult step
- [x] Composer
- [x] Budget per space
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `consult: true` on a branch ask (`?space=NAME&branch=true`): ask.Agent.consult takes the branch scopes the auth middleware issued, picks the teams that run a department agent and whose glossary the question uses or that it names (the SB-4007 rule), at most `ask.MaxConsulted` (6) by terms then name, and asks each one's agent exactly as a direct ask in that team, as the asker, under `ask.AsConsulted` (no branch, no rate take, no conversation, no pin or suggestion, no table tool). `Response.Departments` holds a section per team; `Response.NotConsulted` says why the others weren't (`no_match`, `no_agent`, `paused`, `budget`, `over_limit`) and counts the teams the asker isn't in (`not_member`) without naming them. `consult` inside a consultation, without a branch, or company-wide is EINVAL. Model spend meters to each team's space through the existing withSpace. Web: **Ask each department** beside the branch choice, and a section per team. LAW-27 (`laws/law27.go`, TestLAW27_AConsultationIsTheAskersOwn, TestSelf_LAW27CatchesAWideConsultation) is green. Suite `agent-consult` on the new `evals/fixtures/branch` world; evalcell takes `parent`, `glossary`, `branch` and `consult`, and `consulted` is a new check. The cell test TestDepartmentsWorkTogetherThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
