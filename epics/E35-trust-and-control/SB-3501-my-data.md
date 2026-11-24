---
title: "SB-3501: What the brain keeps about me"
summary: "Only an admin can export a person's data."
---

# SB-3501: What the brain keeps about me

| Field | Value |
|---|---|
| Epic | [E35 Trust and Control](README.md) |
| Wave / slice | W34 / S34.1 |
| Stories | US-35.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-01, LAW-11 |
| Depends on | SB-1903 |
| Status | Open |

## Why

Only an admin can export a person's data.

## Acceptance criteria

- [ ] `GET /v1/me/export` gives the caller their own traces, saved questions, steward questions, settings and token names
- [ ] Cites are checked again as them now; a lost one is left out
- [ ] Audited with no content; nobody else's records appear

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
