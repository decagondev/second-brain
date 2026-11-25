---
title: "SB-4605: Account brief"
summary: "Before a customer call, someone searches tickets, CRM notes and contracts one by one."
---

# SB-4605: Account brief

| Field | Value |
|---|---|
| Epic | [E46 Agents You Can Shape](README.md) |
| Wave / slice | W45 / S45.2 |
| Stories | US-46.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-departments(7)`](../../manual/sb-departments.7.md) |
| Laws | LAW-01, LAW-07, LAW-14 |
| Depends on | SB-610, SB-4202 |
| Status | Done |

## Why

Before a customer call, someone searches tickets, CRM notes and contracts one by one.

## Acceptance criteria

- [x] **Account brief** on an account the person can open returns open tickets, 90 days of notes, documents that name it, renewal date and owners
- [x] With `prep: true`, a meeting with the account's domain gets a brief 30 minutes before
- [x] `personal_data` fields stay masked and nothing is written to the CRM

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `POST /v1/account-brief` (`Legwork.AccountBrief`) in a `sales` space that runs `account-brief` or a `support` space that runs `case-brief`, on a `crm:` document the caller can open now in that space; anything else is `EINVAL` and names nothing. The groups are picked by code from the space's own documents (`accountGroups`): the record, open tickets (kind `tickets` naming the account or its domain, not `IsClosed`), contracts and proposals (by title), and notes first seen in the last 90 days; at most 20 each, newest first, each checked as the caller (`openDocs`). Each group with a document is asked once through `withOnly`, as the caller. The brief is a report (`kind: "account"`) with `account` facts counted by code; `owners` are recomputed from what the reader can still open each time it's shown. `renewal` comes from a `Renewal:` line; HubSpot's companies have none by default, so it shows only where a record's text carries one (`hubspottest` gained a company `domain`). With `prep: true`, `PrepDue` first briefs up to 3 accounts of the person's sales and support spaces whose `Domain:` matches an attendee's (`prepAccounts`, `wire.accountSpaces`). No new agent: the existing `account-brief` and `case-brief` prompts and suites, with an ACL case added. Web: **Account brief** on the Legwork tab, and the counted facts on the brief. Tests: `tests/cell/further_test.go` (counts, a hidden document never named, an account owned by someone else and a company-wide call refused), `ask/further_internal_test.go`, `web/src/further.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
