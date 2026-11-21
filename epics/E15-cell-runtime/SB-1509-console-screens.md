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
| Status | Done |

## Why

The steward proposal and access review APIs exist (SB-1201, SB-1202); their screens don't.

## Acceptance criteria

- [x] Stewards propose a change and decide access reviews in the web UI; operators approve by the hash shown
- [x] The UI tests cover each screen; the WCAG audit stays human

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Screens
- [x] UI tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.5): `web/src/Steward.tsx` is the steward's screen (`/?steward=NAME`): it proposes the space's new definition, refuses anything that isn't JSON before sending it, and shows the proposal's diff, plan and the hash an operator approves, or each reason a proposal is refused; it lists the space's access reviews with each person's `via` and last use, records certify and remove (asking for a removal's reason first), shows the cell's refusal of a decision on your own access, and lists the change requests of a completed review. The admin console gained the open steward proposals, approved only by sending back the hash on screen (and showing the manifest to apply) or rejected with a reason, and every access review, overdue first, with its evidence link. `web/src/steward.test.tsx` and `admin.test.tsx` cover each screen, each with an axe check; the WCAG 2.2 audit with real assistive technology stays human, as for SB-307.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
