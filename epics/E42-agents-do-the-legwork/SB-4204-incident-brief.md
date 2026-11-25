---
title: "SB-4204: Incident brief"
summary: "When a Sev-1 opens, on-call spends the first half hour finding the runbook and the last similar incident."
---

# SB-4204: Incident brief

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.2 |
| Stories | US-42.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-01, LAW-09, LAW-16 |
| Depends on | SB-4006 |
| Status | Done |

## Why

When a Sev-1 opens, on-call spends the first half hour finding the runbook and the last similar incident.

## Acceptance criteria

- [x] A ticket with a label in `incident_labels` (default `sev-1`) drafts one `incident-brief` as the space's first active steward
- [x] The brief cites similar past tickets, matching runbooks, the last 7 days' decisions and changes, and owners
- [x] The ticket's text is data (LAW-09); the draft waits in the steward queue and nothing is posted to the ticket

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Trigger
- [x] Agent and prompt
- [x] Notify
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. sb-space(5) incident_labels (default [sev-1]); the Linear connector writes a Labels: line; ask.IncidentLabels/IsIncident, the incident-brief agent and prompt, DraftIncident; the notify job's incidents part (Cell.newIncidents, runIncident) drafts one brief per ticket and space as the first active steward and tells the stewards (incident). Nothing is posted to the ticket. Tests: TestIncidentLabels, TestIncidentLabelsAndVoC, TestFollowUpThroughARunningCell, agent-incident-brief suite (with an injected title).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
