---
title: "SB-4906: Starter questions"
summary: "New askers don't know what a space can answer."
---

# SB-4906: Starter questions

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.2 |
| Stories | US-49.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-4306 |
| Status | Done |

## Why

New askers don't know what a space can answer.

## Acceptance criteria

- [x] Up to 12 starter questions per space, suggested from saved and helpful asks with no asker's name kept
- [x] Only a steward publishes one; they show on the Answer tab and over MCP
- [x] Each answer is composed fresh as the person; two people with different access get different answers

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Suggestions
- [x] Steward curation
- [x] Answer tab and MCP
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built on the suggested questions of SB-3307, now at most 12. `internal/core/ask/starters.go`: `GET /v1/spaces/NAME/suggested/candidates` offers a space's stewards up to 12 questions: the questions of its verified answers, and questions asked there in the last 90 days whose answers at least 2 people rated, at least 3 in 4 helpful, with how many found them helpful and never who. Only a steward publishes, by saving the list. The MCP server lists each starter as a prompt `SPACE/start-N` (`Stewards.UsableStarters`), and the steward screen shows the candidates with **Add**. Each ask of a starter is an ordinary ask as the person; the cell test checks that two people with different access get different answers. Tested in TestScenariosSourcesAndStartersThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
