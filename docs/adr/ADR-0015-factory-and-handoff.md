---
title: "ADR-0015: The control plane is a factory; custody of a cell can be handed off"
summary: "The control plane is a factory; custody of a cell can be handed off."
---

# ADR-0015: The control plane is a factory; custody of a cell can be handed off

| Status | Date | Deciders |
|---|---|---|
| Accepted (design direction; implementation in W7) | 2026-11-18 | Platform lead, security lead |

## Context

The parent organisation wants one **factory** that produces a Second Brain for each company, runs many of them side by side, and can later **hand a company's Second Brain over** to that company. That might be because the company wants to run it in its own cloud account, or because the company is sold.

v1 already stamps an isolated cell per tenant from one template (ADR-0001). If v1 quietly assumes that every cell lives in the factory's cloud account, under factory-held keys, operated by factory staff, then hand-off becomes a migration project later. That's the bite we want to avoid.

## Decision

1. **Name it.** The control plane, the cell template and the release train together are the **factory**. The factory's own organisation is also a normal tenant (ring 0), never a superuser.
2. **Custody is a first-class record.** Every cell has a custody record in the control-plane registry with three fields: `custody` (`factory`, `tenant` or `detached`), `placement` (which cloud account the cell's resources live in) and `key_custodian` (who can unwrap the cell's root key). v1 only ever writes `custody: factory`, but every component reads these fields instead of assuming them.
3. **Custody changes only through a hand-off workflow**, never by editing the manifest. The workflow needs dual approval (factory and tenant owner) and is modelled in `proofs/tla/Handoff.tla`.
4. **Three custody levels:**

   | Level | Infrastructure in | Root key held by | Releases | Factory access |
   |---|---|---|---|---|
   | `factory` (v1) | Factory account | Factory KMS | Pushed by the factory | Operate and release; break-glass audited |
   | `tenant` | Tenant's own cloud account | Tenant KMS | **Pulled** by the cell, signature-verified | Publish releases only, plus whatever the tenant grants |
   | `detached` | Tenant's account | Tenant KMS | None: off the release train | None |

5. **Seams built in v1** (cheap now, expensive later):

   | Seam | Where | v1 cost |
   |---|---|---|
   | Placement is a parameter of the cell template, behind a `Placement` port; no hard-coded account ids | SB-103, SB-114 | One extra parameter and a test that stamps into two mock accounts |
   | Custody record in the registry; code reads it | SB-105, SB-114 | Three columns and a lint |
   | Envelope key hierarchy: cell root key wraps data keys, so the root can be re-wrapped under another custodian's KMS without re-encrypting data | SB-103, SB-114 | A design choice, not extra code |
   | Cells **pull** signed bundles and **push** attestations; the factory never needs inbound network access to a cell | SB-506, SB-509 | Choose pull over push from the start |
   | Audit anchors signed by a per-cell key, so the chain stays verifiable after hand-off | SB-401, SB-412 | Per-cell signing key instead of one factory key |
   | No factory staff identities inside cells; factory actions arrive as control-plane calls or audited break-glass | SB-106 (already) | None |
   | Control-plane endpoint is configuration with a cell identity certificate; a cell can be re-pointed | SB-105 | One config value |

## Alternatives rejected

| Option | Why not |
|---|---|
| Build hand-off now, in v1 | Derails v1. No company has asked for it yet (Q-09) |
| Ignore hand-off until asked | The v1 shortcuts (factory account ids, one factory signing key, push deploys) would each need rework across every cell |
| Hand-off by exporting data and re-onboarding into a fresh cell | Loses audit continuity, and forces a long dual-running period |
| A shared multi-tenant deployment per company group | Breaks ADR-0001; isolation would depend on query correctness |

## Consequences

- v1 gets about one extra ticket (SB-114) and a few extra acceptance criteria. No v1 wave changes length.
- Two new laws: LAW-17 (hand-off leaves no unlisted factory access) and LAW-18 (the root key is never without a custodian).
- `tenant`-custody cells still follow LAW-06 and LAW-08: they stay on the release train by pulling. Only `detached` cells leave it, and by then they're no longer the factory's responsibility.
- The factory must be able to operate cells it can't reach inbound. That's a good property anyway.
