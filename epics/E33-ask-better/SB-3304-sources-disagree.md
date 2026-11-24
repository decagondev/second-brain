---
title: "SB-3304: Sources disagree"
summary: "When two sources disagree, the answer picks one quietly."
---

# SB-3304: Sources disagree

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.1 |
| Stories | US-33.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | — |
| Status | Done |

## Why

When two sources disagree, the answer picks one quietly.

## Acceptance criteria

- [x] A claim may carry `disagree: true`, kept only when it cites at least two different documents the asker can open
- [x] The UI shows Sources disagree before the claim and chat says it in words
- [x] The citation suite gains a disagreeing and an agreeing case, and they pass

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Answer schema
- [x] UI and chat
- [x] Evals
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

answer.Claim.Disagree, kept by Compose only when answer.TwoDocuments; the answer prompt asks for it; the UI and the Slack and Mattermost apps say Sources disagree; the extractive model joins blocks whose first sentences open alike; the citation suite's cite-008 (disagree) and cite-009 (agree), with the new cites_all and disagrees checks. Tests: TestDisagreeNeedsTwoDocuments, chat apps' disagree_test.go, evals tests.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
