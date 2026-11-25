---
title: "E44 Govern Every Agent"
summary: "Every agent has an owner, real feedback improves its evals, the gate catches a bad definition, and admins see the cost and the risk."
---

# E44: Govern Every Agent

> Every agent has an owner, real feedback improves its evals, the gate catches a bad definition, and admins see the cost and the risk.

**Waves:** W43 · **Tickets:** 6 · **Stories:** 2

Planned 2026-11-24 after W40, with the user. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, and every long run has a ceiling.

## User stories

- **US-44.1**: As an *agent owner* or *platform engineer*, I want feedback turned into eval cases, owners and change notes for each agent, and a lint in the gate, so that agents only get better between releases. Delivered by SB-4401, SB-4402, SB-4403.
- **US-44.2**: As an *admin* or *security lead*, I want signed events out, model tiers and guardrail trends, so that the brain fits our other systems at a sensible cost and I see the risk early. Delivered by SB-4404, SB-4405, SB-4406.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S43.1 | Owners and evals | Feedback becomes evals, agent owners and changes, agent lint in the gate | SB-4401, SB-4402, SB-4403 |
| S43.2 | Out, cost and risk | Signed events out, model tiers, guardrail insights | SB-4404, SB-4405, SB-4406 |

## Exit gate

- [x] Every shipped agent has an owner and passes `sb agents lint`
- [x] An accepted candidate case runs in the next gate, and the asker is never named
- [ ] An event body carries ids only
- [ ] A model tier that fails its evals takes no asks
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4401](SB-4401-feedback-to-evals.md) | W43 / S43.1 | Feedback becomes evals | LAW-01, LAW-06, LAW-07 |
| [SB-4402](SB-4402-agent-owners.md) | W43 / S43.1 | Agent owners and changes | LAW-06, LAW-16 |
| [SB-4403](SB-4403-agent-lint.md) | W43 / S43.1 | Agent lint in the gate | LAW-04, LAW-06, LAW-09 |
| [SB-4404](SB-4404-signed-events.md) | W43 / S43.2 | Signed events out | LAW-07, LAW-11 |
| [SB-4405](SB-4405-model-tiers.md) | W43 / S43.2 | Model tiers | LAW-06, LAW-07 |
| [SB-4406](SB-4406-guardrail-insights.md) | W43 / S43.2 | Guardrail insights | LAW-07, LAW-09 |

## Decisions for the user

Approved by the user on 2026-11-24 ([18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md#decisions-for-the-user)), every recommended decision: all of W41 to W43, in order; events off by default, ids only, the receiver fetches with its own token (SB-4404); a fixed rule routes short plain asks to the small model, each tier gated by its own eval run (SB-4405).
