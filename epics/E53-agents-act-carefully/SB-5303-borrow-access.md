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
| Status | Done |

## Why

A scheduled agent can't use a source only someone other than its approver can read.

## Acceptance criteria

- [x] A run that needs a space its approver can't read can ask a named person who can to lend access for this run, showing the plan, the purpose and who will see the result
- [x] The run reads as the lender for those sources only; the result names the lender and shows only to people who can open every cite
- [x] Lending never carries to the next run, and a declined loan runs without those sources (LAW-31)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Loan request and approval
- [x] Run as the lender for named sources
- [x] LAW-31 law test
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built, with LAW-31. `Stewards.RequestLoan` at `POST /v1/spaces/NAME/loans` puts a `loan` request in the lender's own queue; `Stewards.Lend` at `POST /v1/me/loans/ID/approve|decline` closes it either way, and on approve runs **once** as the lender in the lent space (a probe) and puts a `lent` draft, `lender` named, in the steward's queue. The queue shows a lent draft's text to any steward of the space who can open every cite, checked live (`lentHidden`, `ask.LentVisible`), and hides it from everyone else, the requester included when they can't. The run is the lender's ask in the lent space rather than a mix of both people's access, so nothing in it is anyone's but the lender's. LAW-31 in `laws/law31.go` with `TestSelfLentGateIsCaught`. Tested in TestBorrowAccessThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
