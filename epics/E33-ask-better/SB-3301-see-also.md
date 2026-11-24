---
title: "SB-3301: See also"
summary: "An answer shows only what it cites, so the near-misses the search found are lost."
---

# SB-3301: See also

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.1 |
| Stories | US-33.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | SB-3001 |
| Status | Open |

## Why

An answer shows only what it cites, so the near-misses the search found are lost.

## Acceptance criteria

- [ ] The response's `see_also` lists at most 3 documents retrieval kept for the asker that no claim cites, by title, cite and age, never their text
- [ ] The UI lists them as See also with Show passage and Ask about this document; chat leaves them out
- [ ] A document the asker can't open never appears

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
