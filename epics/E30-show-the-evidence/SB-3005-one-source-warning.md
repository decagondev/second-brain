---
title: "SB-3005: One-source warning"
summary: "An answer resting on one document looks the same as one backed by five."
---

# SB-3005: One-source warning

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.2 |
| Stories | US-30.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | — |
| Status | Open |

## Why

An answer resting on one document looks the same as one backed by five.

## Acceptance criteria

- [ ] When every cite of an answer with at least two claims is one document, the response carries `notice_single` and the UI and chat apps say so first
- [ ] Hidden cites don't count
- [ ] Three claims from one document warn; two documents or a one-claim answer don't

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notice
- [ ] UI and chat apps
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
