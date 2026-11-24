---
title: "SB-3502: Forget my questions"
summary: "Only an admin can erase a person's question history."
---

# SB-3502: Forget my questions

| Field | Value |
|---|---|
| Epic | [E35 Trust and Control](README.md) |
| Wave / slice | W34 / S34.1 |
| Stories | US-35.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-611 |
| Status | Done |

## Why

Only an admin can erase a person's question history.

## Acceptance criteria

- [x] `POST /v1/me/forget` with a typed confirmation erases the caller's traces, conversations and saved questions
- [x] A legal hold still keeps what it holds, and the answer counts what was kept
- [x] The audit row has counts only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. `POST /v1/me/forget` with `confirm: "forget my questions"` runs `retention.Job.Forget`: the caller's feedback, conversations, traces, saved questions and shares, any age, through each store's own Expire selection with everyone else's records counted as held. A record a legal hold covers is kept and counted in `held`; audit rows `forget:USER` hold counts only. Tests: `TestForget`, `TestOwnDataThroughARunningCell`, web `owndata.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
