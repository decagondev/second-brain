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
| Status | Done |

## Why

Only an admin can export a person's data.

## Acceptance criteria

- [x] `GET /v1/me/export` gives the caller their own traces, saved questions, steward questions, settings and token names
- [x] Cites are checked again as them now; a lost one is left out
- [x] Audited with no content; nobody else's records appear

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. `GET /v1/me/export` is the operator's export (SB-1903) with the caller as the user, audited the same way (counts and the SHA-256, never text), then rewritten for the person: each trace is its answer's claims with every cite checked again as them now (a lost cite, and a claim left with no cite, left out and counted in `left_out`); steward questions lose the steward's name and the repeat link; only the person's own stores, plus their live tokens' names. The Questions tab has **Download my data**. Tests: `TestOwnDataThroughARunningCell`, web `owndata.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
