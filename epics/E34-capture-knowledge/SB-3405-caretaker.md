---
title: "SB-3405: Take over a document"
summary: "When an owner leaves, their documents show owner gone but nobody takes them over."
---

# SB-3405: Take over a document

| Field | Value |
|---|---|
| Epic | [E34 Capture Knowledge](README.md) |
| Wave / slice | W33 / S33.2 |
| Stories | US-34.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-11, LAW-12, LAW-19 |
| Depends on | SB-2201 |
| Status | Done |

## Why

When an owner leaves, their documents show owner gone but nobody takes them over.

## Acceptance criteria

- [x] A steward names an active member of the space as caretaker of an owner-gone document; one record, audited
- [x] Who to ask and the owners' note use the caretaker until the source names an active owner
- [x] A deprovisioned caretaker is dropped

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Store and API
- [x] Steward screen and answers
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. `PUT /v1/spaces/NAME/caretakers` and `DELETE …?doc=` name or remove a document's caretaker: a document the space holds, the steward can open, and whose owner at the source is definitely gone; the caretaker an active member of the space. One `cell_docs` record (`caretaker`), audited by the hash of the email. Who to ask, the review list and the owners' monthly note use the caretaker (`"caretaker": true`) only while the source names no active owner; a deprovisioned caretaker is nobody; an access request never goes to a caretaker. Tests: `TestCaretakersAndDefinedTermsThroughARunningCell`, `TestHelpedOwners`, web `caretaker.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
