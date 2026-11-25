---
title: "SB-5302: Try a trigger first"
summary: "A new trigger goes live without anyone seeing what it would have done."
---

# SB-5302: Try a trigger first

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.1 |
| Stories | US-53.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-5301, SB-4704 |
| Status | Open |

## Why

A new trigger goes live without anyone seeing what it would have done.

## Acceptance criteria

- [ ] **Try it** replays the last 14 days of matching events as the approver and shows what it would have drafted
- [ ] It shows what each run would have cost, and nothing is sent
- [ ] Events the approver can't open are counted, not shown

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Replay
- [ ] Cost estimate
- [ ] Tests
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
