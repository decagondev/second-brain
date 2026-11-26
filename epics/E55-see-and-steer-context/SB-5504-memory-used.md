---
title: "SB-5504: Memory used, shown"
summary: "A person can't tell when a remembered line shaped an answer."
---

# SB-5504: Memory used, shown

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.1 |
| Stories | US-55.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-25 |
| Depends on | SB-4303 |
| Status | Open |

## Why

A person can't tell when a remembered line shaped an answer.

## Acceptance criteria

- [ ] An answer that used a memory line shows it under the answer with **Forget this**
- [ ] The line shown is the one in the trace; an answer that used none says nothing
- [ ] Forgetting it removes it from the next answer

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Memory lines on the response
- [ ] Web: **Forget this**
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
