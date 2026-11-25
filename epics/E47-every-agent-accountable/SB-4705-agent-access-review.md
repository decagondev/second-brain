---
title: "SB-4705: Agent access review"
summary: "Agents keep triggers, tools and spaces long after anyone needs them."
---

# SB-4705: Agent access review

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.2 |
| Stories | US-47.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-11, LAW-16 |
| Depends on | SB-3504, SB-4402 |
| Status | Done |

## Why

Agents keep triggers, tools and spaces long after anyone needs them.

## Acceptance criteria

- [x] Every 90 days each agent gets a review of its spaces, triggers and tools with what it used
- [x] Anything unused is proposed off as a manifest diff to the stewards; nothing changes without them
- [x] The review and its outcome are audited

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Job
- [x] Proposal
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `reviews.OpenAgents` opens one agent review a quarter for each space with stewards: its department agents, brief, updates and leader's week, each with its tools and last use (`ask.LastUses` from drafts and traces). Anything not used in 90 days is marked unused, and `propose_off` is the manifest diff `TakeOff` makes; nothing changes until the stewards complete it, and their removals become one manifest proposal (LAW-16). Tools are shown but never proposed off; incident triggers aren't reviewed, since they can't be turned off. The jobsvc `review` job calls `Cell.RunAgentReviews`, which audits each opening; completing it is audited as every review is (LAW-11). Stewards are told when it opens and when it's overdue; the web's review screen labels agent items and shows the proposal. TestOpenAgentsProposesTheUnusedOff and TestProveItThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
