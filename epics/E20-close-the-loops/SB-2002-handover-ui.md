---
title: "SB-2002: Handover brief in the UI"
summary: "W16 built the handover brief as an endpoint only; managers and stewards can't use it without curl."
---

# SB-2002: Handover brief in the UI

| Field | Value |
|---|---|
| Epic | [E20 Close the Loops](README.md) |
| Wave / slice | W19 / S19.1 |
| Stories | US-20.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-12 |
| Depends on | — |
| Status | Open |

## Why

W16 built the handover brief as an endpoint only; managers and stewards can't use it without curl.

## Acceptance criteria

- [ ] The web UI gains a Handover section: enter a person, see the cited brief grouped by location
- [ ] The count of documents the asker can't open is shown with the advice to have access moved before the person leaves
- [ ] A refusal (not their manager, steward or self) shows the policy's reason

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] UI section
- [ ] UI test with axe
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
