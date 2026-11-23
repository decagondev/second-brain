---
title: "E19 Protect the Data"
summary: "Nothing readable leaves the box, the keys can't be lost with the box, and the company can meet a data-protection request without a developer."
---

# E19: Protect the Data

> Nothing readable leaves the box, the keys can't be lost with the box, and the company can meet a data-protection request without a developer.

**Waves:** W18 · **Tickets:** 7 · **Stories:** 3

Proposed 2026-11-22 after W16 and accepted by the user the same day, with every recommended decision. The concept is in [10-run-protect-close.md](../../docs/10-run-protect-close.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, and no new page.

## User stories

- **US-19.1**: As a *data-protection officer*, I want copies off the box to be unreadable without a kit three custodians hold, so that losing the NAS or the box loses nothing to anyone else. Delivered by SB-1901, SB-1902.
- **US-19.2**: As a *data-protection officer*, I want to export a person's data, see personal data in spaces, preview retention and end sessions, so that the company meets its duties to people from the console. Delivered by SB-1903, SB-1904, SB-1905, SB-1906.
- **US-19.3**: As an *auditor*, I want a signed statement when a box is retired, so that the end of a box's data is provable. Delivered by SB-1907.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S18.1 | Off the box | Sealed off-box copies and a 3-of-5 recovery kit | SB-1901, SB-1902 |
| S18.2 | Duties to people | Subject export, personal data flags, retention preview, sign out everywhere | SB-1903, SB-1904, SB-1905, SB-1906 |
| S18.3 | End of life | Retire a box with proof | SB-1907 |

## Exit gate

- [x] Nothing in `SB_OFFSITE` is readable without the recovery key
- [x] Any three of five shares rebuild the keys; two reveal nothing
- [x] Export, erase, export round-trips to empty
- [x] Retention plans match what the job deletes; a revoked session is refused
- [x] A retired box leaves a signed statement and no key
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-1901](SB-1901-sealed-offsite.md) | W18 / S18.1 | Off-box copies sealed | LAW-07, LAW-11 |
| [SB-1902](SB-1902-recovery-kit.md) | W18 / S18.1 | Recovery kit in shares | LAW-18 |
| [SB-1903](SB-1903-subject-export.md) | W18 / S18.2 | Give a person their data | LAW-04, LAW-07 |
| [SB-1904](SB-1904-pii-flags.md) | W18 / S18.2 | Personal data flags at ingest | LAW-07, LAW-13 |
| [SB-1905](SB-1905-retention-preview.md) | W18 / S18.2 | Retention preview | LAW-16 |
| [SB-1906](SB-1906-sign-out-everywhere.md) | W18 / S18.2 | Sign out everywhere | LAW-04, LAW-12 |
| [SB-1907](SB-1907-retire.md) | W18 / S18.3 | Retire a box with proof | LAW-11, LAW-18 |

## Decisions for the user

Decided 2026-11-22 ([10-run-protect-close.md](../../docs/10-run-protect-close.md#decisions-for-the-user)): all of W17 to W19, in order; a 3-of-5 recovery kit on paper (SB-1902); sealing on by default (SB-1901); Mattermost (SB-2005).
