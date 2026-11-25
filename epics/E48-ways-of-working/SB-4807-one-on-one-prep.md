---
title: "SB-4807: 1:1 prep"
summary: "Managers prepare 1:1s by digging through updates, goals and action items by hand."
---

# SB-4807: 1:1 prep

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.2 |
| Stories | US-48.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-01, LAW-12, LAW-30 |
| Depends on | SB-4801, SB-4102, SB-4203 |
| Status | Done |

## Why

Managers prepare 1:1s by digging through updates, goals and action items by hand.

## Acceptance criteria

- [x] The `one-on-one` rhythm drafts a brief before each recurring 1:1 from the report's published updates, goals, action items and waits, checked as the manager
- [x] The report sees exactly the manager's brief and can add a line; it never rates or scores
- [x] LAW-30 is a law test and goes green; a private ask, inbox item, routine or memory entry never appears

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] One-on-one rhythm
- [x] Brief agent and eval suite
- [x] Shared view
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. The built-in `one-on-one` rhythm (12 weekly steps, approver `manager`, drafter `one-on-one`, `internal/core/ask/oneonone.go`) is started by the report's SCIM manager only. Each brief is put together by code as the manager from the goals the report owns (status and waits) and their open action items in the space, each line cited and kept only when both can open its cite; the same text goes to the report's own queue at the same time, and `POST /v1/rhythms/ID/line` lets the report add a line to both copies. No asks, inbox, memory, routines or ratings (LAW-30, `laws/law30.go`, with a self-test that leaks an ask). Briefs follow the rhythm's weekly days, not the calendar's 1:1 events, and the report's space updates are not drawn in (updates are the space's, not the report's). No model writes it, so it needs no eval suite.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
