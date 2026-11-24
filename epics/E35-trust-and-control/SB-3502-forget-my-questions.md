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
| Status | Open |

## Why

Only an admin can erase a person's question history.

## Acceptance criteria

- [ ] `POST /v1/me/forget` with a typed confirmation erases the caller's traces, conversations and saved questions
- [ ] A legal hold still keeps what it holds, and the answer counts what was kept
- [ ] The audit row has counts only

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
