---
title: "SB-4401: Feedback becomes evals"
summary: "An agent's bad answers are rated and reported, but never become tests, so the same mistake ships again."
---

# SB-4401: Feedback becomes evals

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.1 |
| Stories | US-44.1 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-eval(5)`](../../manual/sb-eval.5.md) |
| Laws | LAW-01, LAW-06, LAW-07 |
| Depends on | SB-2904 |
| Status | Done |

## Why

An agent's bad answers are rated and reported, but never become tests, so the same mistake ships again.

## Acceptance criteria

- [x] An unhelpful or reported answer on a named agent becomes a candidate case in its owner's inbox, with the asker's name removed
- [x] The owner accepts it with the expected answer or cites, or dismisses it; accepted cases run in the next gate
- [x] A candidate on documents the owner can't open can't be accepted

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Records
- [x] API
- [x] UI
- [x] Gate
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Stewards.CaseCandidates: down-rated or reported answers of ask and the department agents in the last 30 days, for the people the bundle names as owner, the question from the conversation and each claim shown only if the owner can open every cite (the rest counted in hidden). DecideCase accepts with an expect the owner can open now, or dismisses; both audited (access, eval_case). EvalCase gains agent, from and dismissed; the evals job runs a case with its agent (Agent.ProbeAs) and never a dismissed one. GET /v1/agents/cases, POST /v1/agents/cases/{id}; inbox kind case. Web: Candidate cases on the Inbox tab. TestCasesFromFeedbackThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
