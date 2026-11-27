---
title: "E62 Reach Confluence and Jira"
summary: "A team on Confluence and Jira gets the same brain as one on Google Drive and Linear, read-only, with Atlassian's own permissions behind every check."
---

# E62: Reach Confluence and Jira

> A team on Confluence and Jira gets the same brain as one on Google Drive and Linear, read-only, with Atlassian's own permissions behind every check.

**Waves:** W61 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-27 after W58, with the user. The concept is in [24-prove-read-reach.md](../../docs/24-prove-read-reach.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, lent access lasts one run and travels with its result, a label never drops, a model call stays in its region, a new vendor is an adapter against a recorded fake, and text the brain didn't read from the source itself carries the AI mark.

## User stories

- **US-62.1**: As an *admin* or *anyone*, I want Confluence with its restrictions and Atlassian groups behind every check, so that a team's wiki is cited exactly as Confluence allows. Delivered by SB-6201, SB-6202, SB-6203.
- **US-62.2**: As *anyone* or a *steward*, I want Jira issues, work events, service requests and drafted issues, so that the brain works where the team tracks its work. Delivered by SB-6204, SB-6205, SB-6206, SB-6207.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S61.1 | Confluence and groups | Confluence, Atlassian groups and people, Check the Atlassian setup | SB-6201, SB-6202, SB-6203 |
| S61.2 | Jira | Jira, Jira as work events, Start on a service request, Draft a Jira issue | SB-6204, SB-6205, SB-6206, SB-6207 |

## Exit gate

- [ ] The recorded Confluence fake syncs with its restrictions, and a restricted page is cited only to its group
- [ ] An issue under a security level is cited only to people in it; a deprovisioned person loses access within 5 minutes
- [ ] Only a service queue a space names is read, nothing is sent or filed without a person, and a token with a write scope fails the setup check
- [ ] Every law test still green; no new store, service or Go module; real-site runs stay In review until `sb prove`

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-6201](SB-6201-confluence.md) | W61 / S61.1 | Confluence | LAW-01, LAW-13 |
| [SB-6202](SB-6202-atlassian-groups.md) | W61 / S61.1 | Atlassian groups and people | LAW-01, LAW-03, LAW-12 |
| [SB-6203](SB-6203-check-atlassian.md) | W61 / S61.1 | Check the Atlassian setup | LAW-03, LAW-16 |
| [SB-6204](SB-6204-jira.md) | W61 / S61.2 | Jira | LAW-01, LAW-13 |
| [SB-6205](SB-6205-jira-events.md) | W61 / S61.2 | Jira as work events | LAW-09, LAW-16 |
| [SB-6206](SB-6206-service-requests.md) | W61 / S61.2 | Start on a service request | LAW-09, LAW-16 |
| [SB-6207](SB-6207-draft-jira-issue.md) | W61 / S61.2 | Draft a Jira issue | LAW-16, LAW-32 |

## Decisions for the user

Approved by the user on 2026-11-27 ([24-prove-read-reach.md](../../docs/24-prove-read-reach.md#decisions-for-the-user)), every recommended decision.
