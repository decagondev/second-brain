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
| Status | Open |

## Why

The UI has never been checked for people using a screen reader or a keyboard alone.

## Acceptance criteria

- [ ] `make web` runs an automated WCAG 2.2 AA check over every tab and fails on any serious finding
- [ ] Ask, cite, review and inbox each pass a keyboard-only test
- [ ] Cites, labels and notices have screen reader names, and colour is never the only sign of a label or a stale cite

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Accessibility check in `make web`
- [ ] Keyboard-only tests and screen reader names
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
