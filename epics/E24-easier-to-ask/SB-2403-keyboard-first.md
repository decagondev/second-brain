---
title: "SB-2403: Keyboard first"
summary: "Asking needs the mouse more than it should."
---

# SB-2403: Keyboard first

| Field | Value |
|---|---|
| Epic | [E24 Easier to Ask](README.md) |
| Wave / slice | W23 / S23.1 |
| Stories | US-24.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-14 |
| Depends on | — |
| Status | Done |

## Why

Asking needs the mouse more than it should.

## Acceptance criteria

- [x] `/` focuses the ask box, Ctrl or Cmd+Enter asks, Esc closes a panel and returns focus, and a skip link comes first
- [x] No shortcut fires while typing in a field; a Keyboard help line lists them
- [x] A test drives ask, follow-up, save and the tabs with the keyboard only; axe in each language

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Shortcuts and focus
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S23.1). A "Skip to the conversation" link comes first; `/` goes to the ask box (and the Answer tab), Ctrl or Cmd+Enter in the ask box asks, Esc closes the API tokens panel and returns focus to its button; none fires while typing in a field. A line under the ask box lists them, tied to it by `aria-describedby`. Tests drive the keys and run axe in English, German and French.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
