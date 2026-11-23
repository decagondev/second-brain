---
title: "E27 Fresh Answers"
summary: "An asker can tell at a glance when an answer rests on old sources, and the brain prefers the newer of two equal ones."
---

# E27: Fresh Answers

> An asker can tell at a glance when an answer rests on old sources, and the brain prefers the newer of two equal ones.

**Waves:** W26 · **Tickets:** 6 · **Stories:** 2

Proposed 2026-11-23 after W25 and accepted by the user the same day, with every recommended decision. The concept is in [13-fresh-tidy-focused.md](../../docs/13-fresh-tidy-focused.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-27.1**: As an *asker*, I want to be warned when an answer rests on old or unsynced sources, and to report one that is outdated, so that I don't rely on something that moved. Delivered by SB-2701, SB-2702, SB-2703, SB-2704.
- **US-27.2**: As a *steward*, I want a document's own review date honoured, and newer documents preferred in a tie, so that the brain's idea of fresh matches ours. Delivered by SB-2705, SB-2706.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S26.1 | Say when it's old | Old-source warning, reviewed on the cite, report as outdated, sync lag notice | SB-2701, SB-2702, SB-2703, SB-2704 |
| S26.2 | Trust the dates | Review-by dates in the document, newer wins a tie | SB-2705, SB-2706 |

## Exit gate

- [x] An answer resting mostly on stale cites warns, and a hidden stale document changes nothing
- [x] A reviewed cite isn't stale until its review lapses
- [x] An outdated report lists the document, and a mark clears it
- [x] A connector past its freshness target is named on the answer; an unknown sync time is never shown as fresh
- [x] A date in a document only makes it stale sooner
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2701](SB-2701-old-source-warning.md) | W26 / S26.1 | Old-source warning | LAW-01, LAW-14 |
| [SB-2702](SB-2702-reviewed-on-cite.md) | W26 / S26.1 | Reviewed shows on the cite | LAW-13 |
| [SB-2703](SB-2703-report-outdated.md) | W26 / S26.1 | Report as outdated | LAW-01 |
| [SB-2704](SB-2704-sync-lag-notice.md) | W26 / S26.1 | Sync lag notice | LAW-03 |
| [SB-2705](SB-2705-review-by-dates.md) | W26 / S26.2 | Review-by dates in the document | LAW-09, LAW-13 |
| [SB-2706](SB-2706-newer-wins-tie.md) | W26 / S26.2 | Newer wins a tie | LAW-01 |

## Decisions for the user

Decided 2026-11-23 ([13-fresh-tidy-focused.md](../../docs/13-fresh-tidy-focused.md#decisions-for-the-user)): all of W26 to W28, in order; warn the asker when more than half of an answer's cites are stale (SB-2701); stewards retire a document from their own space's answers only, undoable and audited (SB-2801); owners' monthly note on by default, each person can turn it off (SB-2806); conversations send the last 8 questions (SB-2901).
