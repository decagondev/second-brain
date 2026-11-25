---
title: "SB-4604: Post-incident review"
summary: "After an incident, someone rebuilds the timeline by hand and follow-ups get lost."
---

# SB-4604: Post-incident review

| Field | Value |
|---|---|
| Epic | [E46 Agents You Can Shape](README.md) |
| Wave / slice | W45 / S45.2 |
| Stories | US-46.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-01, LAW-09, LAW-16 |
| Depends on | SB-4204, SB-4203 |
| Status | Open |

## Why

After an incident, someone rebuilds the timeline by hand and follow-ups get lost.

## Acceptance criteria

- [ ] A closed ticket that had an incident brief drafts one review to the steward queue
- [ ] The review holds a timeline, decisions, changes and follow-ups as action items the steward confirms
- [ ] An instruction in the ticket is ignored; nothing is posted to the ticket

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent and prompt
- [ ] Trigger
- [ ] Eval suite
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
