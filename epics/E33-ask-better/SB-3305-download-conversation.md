---
title: "SB-3305: Download the conversation"
summary: "A conversation can be copied one answer at a time, not as a whole."
---

# SB-3305: Download the conversation

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.2 |
| Stories | US-33.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-14 |
| Depends on | SB-2401 |
| Status | Open |

## Why

A conversation can be copied one answer at a time, not as a whole.

## Acceptance criteria

- [ ] Download conversation saves every turn as one Markdown file made in the browser, by SB-2401's Copy rules
- [ ] Nothing is sent to the cell
- [ ] Uncited claims keep their label; hidden sources are only the screen's count

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
