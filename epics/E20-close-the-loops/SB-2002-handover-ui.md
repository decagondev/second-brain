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
| Status | Done |

## Why

W16 built the handover brief as an endpoint only; managers and stewards can't use it without curl.

## Acceptance criteria

- [x] The web UI gains a Handover section: enter a person, see the cited brief grouped by location
- [x] The count of documents the asker can't open is shown with the advice to have access moved before the person leaves
- [x] A refusal (not their manager, steward or self) shows the policy's reason

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI section
- [x] UI test with axe
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built 2026-11-23. The web UI's **Handover** tab (`web/src/Handover.tsx`) takes a person's user id, posts `POST /v1/handover`, and shows the brief grouped by location (the response's new `locations`, cite to location, for documents the asker can open only), each line through the same `ClaimView` as an answer (LAW-14). `hidden` is shown as a count with the advice to have access moved before the person leaves; `dropped` as a count. A refusal shows the policy's reason, which the cell now gives in its `EPERM` message. Tests: `web/src/handover.test.tsx` (grouping, counts, refusal, axe clean); `tests/cell/handover_test.go` checks `locations` and the reason. Manual: sb-ask(3).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
