---
title: "SB-2304: Search results while the answer is written"
summary: "On a CPU box answers take 7 to 32 seconds with nothing on screen."
---

# SB-2304: Search results while the answer is written

| Field | Value |
|---|---|
| Epic | [E23 Fit the Site](README.md) |
| Wave / slice | W22 / S22.2 |
| Stories | US-23.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | — |
| Status | Open |

## Why

On a CPU box answers take 7 to 32 seconds with nothing on screen.

## Acceptance criteria

- [ ] The web UI asks search and ask together and shows the allowed sources (title, owner, age) at once, then the cited answer
- [ ] The sources are search's, checked as the asker, and are never shown as claims
- [ ] With a slow fake model the sources render first; axe passes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
