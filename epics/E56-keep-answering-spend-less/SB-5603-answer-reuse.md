---
title: "SB-5603: Answered before, checked now"
summary: "The same question asked twice in a day costs twice and can come back different."
---

# SB-5603: Answered before, checked now

| Field | Value |
|---|---|
| Epic | [E56 Keep Answering, Spend Less](README.md) |
| Wave / slice | W55 / S55.1 |
| Stories | US-56.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-4606, SB-4201 |
| Status | Open |

## Why

The same question asked twice in a day costs twice and can come back different.

## Acceptance criteria

- [ ] An ask whose normalised question, space, scope and language match one answered in the last 7 days reuses it when every cite's document is unchanged and the asker can open all of them now
- [ ] A reused answer says "answered on DATE, checked now", offers **Ask fresh** and costs no model call
- [ ] Memory, exclusions, kept or dropped documents, a file or a collection make an ask unique and never reuse

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Reuse lookup from traces
- [ ] Recheck as the asker
- [ ] Web: **Ask fresh**
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
