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
| Status | Open |

## Why

Before a customer call, someone searches tickets, CRM notes and contracts one by one.

## Acceptance criteria

- [ ] **Account brief** on an account the person can open returns open tickets, 90 days of notes, documents that name it, renewal date and owners
- [ ] With `prep: true`, a meeting with the account's domain gets a brief 30 minutes before
- [ ] `personal_data` fields stay masked and nothing is written to the CRM

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent and prompt
- [ ] UI
- [ ] Eval suite
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
