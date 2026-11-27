---
title: "SB-6107: Usable by everyone"
summary: "The UI has never been checked for people using a screen reader or a keyboard alone."
---

# SB-6107: Usable by everyone

| Field | Value |
|---|---|
| Epic | [E61 Every Format, Every Language](README.md) |
| Wave / slice | W60 / S60.2 |
| Stories | US-61.2 |
| Manual | [`sb-tour(7)`](../../manual/sb-tour.7.md) |
| Laws | none new |
| Depends on | SB-2401, SB-2403 |
| Status | Done |

## Why

The UI has never been checked for people using a screen reader or a keyboard alone.

## Acceptance criteria

- [x] `make web` runs an automated WCAG 2.2 AA check over every tab and fails on any serious finding
- [x] Ask, cite, review and inbox each pass a keyboard-only test
- [x] Cites, labels and notices have screen reader names, and colour is never the only sign of a label or a stale cite

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Accessibility check in `make web`
- [x] Keyboard-only tests and screen reader names
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `web/src/accessible.test.tsx` runs axe (WCAG 2.0 to 2.2 A and AA tags) over every tab of the app and over an answer with a label, a stale cite, a machine-read cite and a notice, failing on a serious or critical violation; it checks the tabs move with the arrow keys, asking works with Ctrl+Enter, and cites and Show passage are reachable by keyboard; `web/src/contrast.test.ts` checks the stylesheet's colour tokens meet AA in both schemes, since jsdom computes no colour. The check runs in `make web` with jsdom rather than Playwright, which the web suite doesn't use.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
