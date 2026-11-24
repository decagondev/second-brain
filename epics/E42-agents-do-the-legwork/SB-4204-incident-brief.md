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
| Status | Open |

## Why

When a Sev-1 opens, on-call spends the first half hour finding the runbook and the last similar incident.

## Acceptance criteria

- [ ] A ticket with a label in `incident_labels` (default `sev-1`) drafts one `incident-brief` as the space's first active steward
- [ ] The brief cites similar past tickets, matching runbooks, the last 7 days' decisions and changes, and owners
- [ ] The ticket's text is data (LAW-09); the draft waits in the steward queue and nothing is posted to the ticket

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Trigger
- [ ] Agent and prompt
- [ ] Notify
- [ ] Eval suite
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
