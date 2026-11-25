---
title: "E53 Agents That Act Carefully"
summary: "Agents start on the work events that matter, can be tried before they're turned on, can borrow access for one run with a named lender, and draft the next action for a person to send."
---

# E53: Agents That Act Carefully

> Agents start on the work events that matter, can be tried before they're turned on, can borrow access for one run with a named lender, and draft the next action for a person to send.

**Waves:** W52 · **Tickets:** 6 · **Stories:** 2

Planned 2026-11-25 after W49, with the user. The concept is in [21-guard-write-act.md](../../docs/21-guard-write-act.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, and lent access lasts one run and travels with its result.

## User stories

- **US-53.1**: As an *agent owner* or *lender*, I want triggers on work events, a replay before they go live and access lent for one run, so that agents start at the right moment without seeing more than they should. Delivered by SB-5301, SB-5302, SB-5303.
- **US-53.2**: As *anyone*, an *admin* or a *meeting attendee*, I want drafted next actions, managed MCP access and meeting transcripts from Drive, so that work moves on without the brain acting on its own. Delivered by SB-5304, SB-5305, SB-5306.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S52.1 | Start on work | Start on a work event, Try a trigger first, Borrow access for one run | SB-5301, SB-5302, SB-5303 |
| S52.2 | Do the next thing | Draft the next action, Managed MCP access, Meeting transcripts from Drive | SB-5304, SB-5305, SB-5306 |

## Exit gate

- [ ] A work event starts one draft, floods batch, and nothing publishes without the approver
- [ ] Lent access lasts one run, names the lender, and its result reaches only people who can open every cite (LAW-31 green)
- [ ] Nothing is sent to Linear, Slack or Mattermost without the person's press
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5301](SB-5301-work-events.md) | W52 / S52.1 | Start on a work event | LAW-01, LAW-16, LAW-29 |
| [SB-5302](SB-5302-try-a-trigger.md) | W52 / S52.1 | Try a trigger first | LAW-01, LAW-07 |
| [SB-5303](SB-5303-borrow-access.md) | W52 / S52.1 | Borrow access for one run | LAW-01, LAW-26, LAW-31 |
| [SB-5304](SB-5304-draft-next-action.md) | W52 / S52.2 | Draft the next action | LAW-04, LAW-16 |
| [SB-5305](SB-5305-managed-mcp.md) | W52 / S52.2 | Managed MCP access | LAW-01, LAW-07 |
| [SB-5306](SB-5306-meet-transcripts.md) | W52 / S52.2 | Meeting transcripts from Drive | LAW-01, LAW-02 |

## Decisions for the user

Approved by the user on 2026-11-25 ([21-guard-write-act.md](../../docs/21-guard-write-act.md#decisions-for-the-user)), every recommended decision: borrowed access is included, one run at a time, with a named lender and the result limited to people who can open every cite (SB-5303, LAW-31); drafted actions go to Linear and Slack or Mattermost and are sent by the person with their own token (SB-5304); meeting transcripts come from Google Meet through the Drive connector, no new vendor (SB-5306).
