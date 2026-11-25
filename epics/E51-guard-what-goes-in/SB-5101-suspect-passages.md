---
title: "SB-5101: Suspect passages"
summary: "The sanitiser treats documents as data at ingest, but a passage that reads like instructions to the model is never shown to anyone."
---

# SB-5101: Suspect passages

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.1 |
| Stories | US-51.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-09, LAW-11 |
| Depends on | SB-207, SB-4406 |
| Status | Open |

## Why

The sanitiser treats documents as data at ingest, but a passage that reads like instructions to the model is never shown to anyone.

## Acceptance criteria

- [ ] A retrieved passage addressed to the model is skipped, never quoted, and the answer says how many were skipped
- [ ] Stewards see suspect passages on the review list (`section=suspect`) and the owner gets one notice per document
- [ ] `suspect: monitor` logs the same events without skipping; the default is `skip`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Passage check at read time
- [ ] Review list section and owner notice
- [ ] Monitor setting
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
