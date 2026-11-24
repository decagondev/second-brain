---
title: "SB-4207: Goal check-ins"
summary: "Goals go quiet: their status is set once a quarter and nobody notices when the work behind them stops."
---

# SB-4207: Goal check-ins

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.2 |
| Stories | US-42.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-16, LAW-24 |
| Depends on | SB-4101 |
| Status | Open |

## Why

Goals go quiet: their status is set once a quarter and nobody notices when the work behind them stops.

## Acceptance criteria

- [ ] Each Monday, a goal gets a drafted check-in for its owner from the tickets, decisions, updates and documents that cite it, with a suggested status
- [ ] The owner publishes it as a note under the goal or discards it; nothing is published without them
- [ ] A goal with no published check-in for 3 weeks is flagged on its owner's inbox and its space's home; only counts roll up

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent and prompt
- [ ] Notify
- [ ] Notes
- [ ] UI
- [ ] Eval suite
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
