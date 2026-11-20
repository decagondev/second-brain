---
title: "SB-1509: Console screens for stewards and access reviews"
summary: "The steward proposal and access review APIs exist (SB-1201, SB-1202); their screens don't."
---

# SB-1509: Console screens for stewards and access reviews

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.5 |
| Stories | US-15.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-15, LAW-16 |
| Depends on | [SB-1502](SB-1502-api-service.md) |
| Status | Open |

## Why

The steward proposal and access review APIs exist (SB-1201, SB-1202); their screens don't.

## Acceptance criteria

- [ ] Stewards propose a change and decide access reviews in the web UI; operators approve by the hash shown
- [ ] The UI tests cover each screen; the WCAG audit stays human

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Screens
- [ ] UI tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
