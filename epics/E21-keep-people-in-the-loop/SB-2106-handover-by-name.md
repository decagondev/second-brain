---
title: "SB-2106: Handover by name"
summary: "The Handover tab takes a user id, which managers don't know."
---

# SB-2106: Handover by name

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.3 |
| Stories | US-21.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

The Handover tab takes a user id, which managers don't know.

## Acceptance criteria

- [x] The Handover tab gains a person search listing, by name and email, only the people the asker may brief: themselves, their reports, members of spaces they steward
- [x] A colleague with no relation isn't in the list, whatever they type
- [x] Typing a user id still works

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] People endpoint
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: Done. `GET /v1/handover/people?q=` (`ask.Agent.HandoverPeople`) looks among the caller, the people they manage and the members of spaces they steward (`wire.Cell.briefable`), keeps active ones whose user name or email matches, and keeps each only if the same `brief_handover` policy decision allows it now. The Handover tab takes a name, an email or a user id: one match briefs at once, several are listed to pick from. `TestLAW01_HandoverSearchListsOnlyWhomTheAskerMayBrief` with a self-test that catches a search skipping the policy; the running-cell handover test shows a colleague finds no one.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
