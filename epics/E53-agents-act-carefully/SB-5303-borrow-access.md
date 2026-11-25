---
title: "SB-5303: Borrow access for one run"
summary: "A scheduled agent can't use a source only someone other than its approver can read."
---

# SB-5303: Borrow access for one run

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.1 |
| Stories | US-53.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-01, LAW-26, LAW-31 |
| Depends on | SB-4006, SB-4201 |
| Status | Open |

## Why

A scheduled agent can't use a source only someone other than its approver can read.

## Acceptance criteria

- [ ] A run that needs a space its approver can't read can ask a named person who can to lend access for this run, showing the plan, the purpose and who will see the result
- [ ] The run reads as the lender for those sources only; the result names the lender and shows only to people who can open every cite
- [ ] Lending never carries to the next run, and a declined loan runs without those sources (LAW-31)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Loan request and approval
- [ ] Run as the lender for named sources
- [ ] LAW-31 law test
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
