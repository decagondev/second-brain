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
| Status | Done |

## Why

People on Microsoft 365 can't ask the brain where they work.

## Acceptance criteria

- [x] `internal/adapters/chatapp/teamsapp` and `POST /chat/teams`: mentions and DMs, cites, follow-ups in the reply chain
- [x] Incoming activities verified against Microsoft's signing keys with the standard library; anything else refused
- [x] A confidential answer goes to the person in a DM

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Teams app adapter
- [x] Activity verification
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `internal/adapters/chatapp/teamsapp` at `POST /chat/teams`: the Bot Framework's RS256 token checked with the standard library against its published keys (issuer, this app id, times, the activity's serviceUrl), the sender's Entra id mapped through Graph and the Slack app's rules, a channel mapped to its space by name, and the answer always in the person's personal chat (Teams has no message only one person sees), the reply chain getting a line; follow-ups in a reply chain are one conversation per person (`Request.Thread`). sb-cell(5) `teams` and `vault://tenants/TENANT/teams`. Tested in TestAskInTeams against a recorded Bot Framework.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
