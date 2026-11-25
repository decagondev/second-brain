---
title: "SB-4505: Decisions that replace decisions"
summary: "An answer can cite a decision that a later one replaced, and nothing says so."
---

# SB-4505: Decisions that replace decisions

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.2 |
| Stories | US-45.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | SB-3702, SB-3605 |
| Status | Done |

## Why

An answer can cite a decision that a later one replaced, and nothing says so.

## Acceptance criteria

- [x] A decision can name the one it replaces, in front matter or when the steward confirms
- [x] A cite of a replaced decision shows the newer one if the asker can open it
- [x] A topic shows its trail of decisions; a suggested replacement waits for the steward

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Front matter
- [x] Composer
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `supersedes:` in a decision's front matter, as KIND:ID or the older decision's title in its space (`supersedeLinks`). `Agent.markReplacedBy` sets `answer.Cite.ReplacedBy` on a cited decision when the asker can open the newer one now, following the chain up to 5 and stopping at the first they can't (`chainNewest`). A new transcript's decisions draft lists `supersede`: documents its claims marked as disagreeing cite, for the steward it ran as; publishing with `supersedes` (`NoteRequest.Supersedes`, checked by `checkSupersedes`) writes `type: decision` and `supersedes:` into the note. `GET /v1/spaces/NAME/decisions?topic=` (`Stewards.Trail`) lists the topic's decisions oldest first, each after the one it replaces (`inOrder`), with owners, the rest counted. Chip **Replaced by**, **Does this replace an earlier decision?** on the draft, and the trail on the home. TestSupersedeLinksAndOrder and the cell test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
