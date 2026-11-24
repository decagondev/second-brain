---
title: "E32 Run Without Surprises"
summary: "The operator learns about trouble from the cell before users do, and can fix the small things in one command."
---

# E32: Run Without Surprises

> The operator learns about trouble from the cell before users do, and can fix the small things in one command.

**Waves:** W31 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W28 and accepted by the user the same day, with every recommended decision. The concept is in [14-evidence-gaps-surprises.md](../../docs/14-evidence-gaps-surprises.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-32.1**: As an *operator* or *admin*, I want to know why a sync fails, refresh one document, alert from Prometheus, and filter the audit by person and date, so that I fix things before anyone asks. Delivered by SB-3201, SB-3202, SB-3203, SB-3204.
- **US-32.2**: As an *operator* or *admin*, I want to see answer time and disk runway coming, and tell everyone about downtime, so that nobody is surprised. Delivered by SB-3205, SB-3206, SB-3207.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S31.1 | Know why | Why a sync fails, refresh one document, health in Prometheus, audit by person and date | SB-3201, SB-3202, SB-3203, SB-3204 |
| S31.2 | See it coming | Answer time watch, disk runway, notice to everyone | SB-3205, SB-3206, SB-3207 |

## Exit gate

- [ ] A sync failure is classified and its record holds no URL
- [ ] An unreadable health line is never 0 on both gauges
- [ ] A disk runway is never shown from too little history
- [ ] A notice ends by itself within 7 days
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3201](SB-3201-sync-reason.md) | W31 / S31.1 | Why a sync fails | LAW-07, LAW-03 |
| [SB-3202](SB-3202-refresh-document.md) | W31 / S31.1 | Refresh one document | LAW-11, LAW-13 |
| [SB-3203](SB-3203-health-metrics.md) | W31 / S31.1 | Health in Prometheus | LAW-03 |
| [SB-3204](SB-3204-audit-filters.md) | W31 / S31.1 | Audit by person and date | LAW-11 |
| [SB-3205](SB-3205-answer-time.md) | W31 / S31.2 | Answer time watch | LAW-07 |
| [SB-3206](SB-3206-disk-runway.md) | W31 / S31.2 | Disk runway | LAW-03 |
| [SB-3207](SB-3207-notice-to-everyone.md) | W31 / S31.2 | Notice to everyone | LAW-09, LAW-11 |

## Decisions for the user

Decided 2026-11-24 ([14-evidence-gaps-surprises.md](../../docs/14-evidence-gaps-surprises.md#decisions-for-the-user)): all of W29 to W31, in order; answers in the language picked in the UI, or the manifest's in chat (SB-3003); passages of up to 600 characters (SB-3001); owners mark only their own documents reviewed (SB-3106); admins alone set the notice, ending within 7 days (SB-3207).
