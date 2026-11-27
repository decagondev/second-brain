---
title: "SB-5904: Ask in Teams"
summary: "People on Microsoft 365 can't ask the brain where they work."
---

# SB-5904: Ask in Teams

| Field | Value |
|---|---|
| Epic | [E59 Reach Microsoft 365](README.md) |
| Wave / slice | W58 / S58.2 |
| Stories | US-59.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-32 |
| Depends on | SB-5801, SB-5405, SB-5903 |
| Status | Open |

## Why

People on Microsoft 365 can't ask the brain where they work.

## Acceptance criteria

- [ ] `internal/adapters/chatapp/teamsapp` and `POST /chat/teams`: mentions and DMs, cites, follow-ups in the reply chain
- [ ] Incoming activities verified against Microsoft's signing keys with the standard library; anything else refused
- [ ] A confidential answer goes to the person in a DM

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Teams app adapter
- [ ] Activity verification
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
