---
title: "E51 Guard What Goes In"
summary: "Nothing a source says can steer an agent or leak through it, a new feature reaches a pilot before everyone, and anyone can see how an agent changed."
---

# E51: Guard What Goes In

> Nothing a source says can steer an agent or leak through it, a new feature reaches a pilot before everyone, and anyone can see how an agent changed.

**Waves:** W50 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-25 after W49, with the user. The concept is in [21-guard-write-act.md](../../docs/21-guard-write-act.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, and lent access lasts one run and travels with its result.

## User stories

- **US-51.1**: As a *steward*, *security reviewer* or *auditor*, I want suspect passages skipped, sensitive data masked on every way out and guardrail events exported, so that no source can steer an agent or leak through it. Delivered by SB-5101, SB-5102, SB-5103.
- **US-51.2**: As an *admin*, *agent owner* or *steward*, I want an agent's history, branch moderators, feature rollouts and a pause link on every notice, so that changes reach a pilot first and anyone allowed can stop an agent. Delivered by SB-5104, SB-5105, SB-5106, SB-5107.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S50.1 | Check what goes in and out | Suspect passages, Nothing sensitive goes out, Guardrail export | SB-5101, SB-5102, SB-5103 |
| S50.2 | Roll out with care | Agent history, Branch moderators, Feature rollouts, Pause from the notice | SB-5104, SB-5105, SB-5106, SB-5107 |

## Exit gate

- [ ] Every injection fixture is skipped on every path (ask, research, briefs, MCP), and the steward sees it
- [ ] A key planted in a source never appears on any way out, and every way out goes through the one filter
- [ ] A restore is a proposal that passes the gate again, and a pilot feature is off for non-members
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5101](SB-5101-suspect-passages.md) | W50 / S50.1 | Suspect passages | LAW-09, LAW-11 |
| [SB-5102](SB-5102-nothing-sensitive-out.md) | W50 / S50.1 | Nothing sensitive goes out | LAW-07, LAW-11 |
| [SB-5103](SB-5103-guardrail-export.md) | W50 / S50.1 | Guardrail export | LAW-07, LAW-11 |
| [SB-5104](SB-5104-agent-history.md) | W50 / S50.2 | Agent history | LAW-06, LAW-11 |
| [SB-5105](SB-5105-branch-moderators.md) | W50 / S50.2 | Branch moderators | LAW-10, LAW-24 |
| [SB-5106](SB-5106-feature-rollouts.md) | W50 / S50.2 | Feature rollouts | LAW-06, LAW-08 |
| [SB-5107](SB-5107-pause-from-notice.md) | W50 / S50.2 | Pause from the notice | LAW-10 |

## Decisions for the user

Approved by the user on 2026-11-25 ([21-guard-write-act.md](../../docs/21-guard-write-act.md#decisions-for-the-user)), every recommended decision: suspect passages are skipped and logged by default, and a space may choose monitor only (SB-5101); a pilot group first, then on for everyone after 14 days unless an admin holds it (SB-5106).
