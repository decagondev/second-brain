---
title: "SB-3303: Short or full answer"
summary: "Every answer is the same length."
---

# SB-3303: Short or full answer

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.1 |
| Stories | US-33.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-14 |
| Depends on | — |
| Status | Open |

## Why

Every answer is the same length.

## Acceptance criteria

- [ ] `POST /v1/ask` takes `length: short`, which adds one prompt line asking for at most 3 claims and keeps at most 3
- [ ] The UI's Short answer toggle is kept in the browser
- [ ] Any other value changes nothing; each claim is still cited or marked uncited

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
