---
title: Problem Statement
summary: What Second Brain solves, for whom, and how we'll know it worked.
---

# Problem statement

| Version | Date | Owner | Status |
|---|---|---|---|
| 0.2 | 2026-11-18 | Platform lead | Draft for review |

## The problem in one paragraph

When a parent organisation acquires a company, that company has no single index of what it knows. It has no reliable record of who may see what, and no record of who did what. Tribal knowledge is spread across docs, chat, tickets and CRM, and it walks out of the door with leavers. Teams adopt AI tools one by one, pasting company data into whatever is to hand, with no permissions, no audit and no way to switch it off. And every acquisition starts again from zero, so the parent pays for the same lesson in every company.

## Who has the problem

| Persona | Today | What they need |
|---|---|---|
| **Asker**: company staff | Search five tools, ask a colleague, wait | A cited answer in seconds, from sources they're allowed to see |
| **Operator**: company IT | Asked to "do AI" with no platform, budget or guardrails | One file and three commands to go live, plus a kill switch |
| **Approver**: company leadership | Can't see what AI tools are doing with company data | To approve consequential actions with the exact payload in front of them |
| **Maintainer**: parent-org platform team | Rebuilds integrations per acquisition | One template, one release train, no per-company code |
| **Auditor / security** | No evidence trail | Tamper-evident proof of who saw and did what, and with which build |

## Why existing approaches fall short

| Approach | Where it breaks |
|---|---|
| Each company buys its own AI tools | No reuse, inconsistent controls, the parent can't support twelve stacks |
| A shared, multi-tenant index for all companies | One bug leaks one company's data to another; regulators and acquired staff won't accept it |
| Copying permissions into the index and trusting them | The copy goes stale. A permission revoked at 09:00 is still "allowed" at 09:10 |
| Letting the model decide what to cite and whether it acted | Models can claim citations and successes that never happened |
| Building per-company customisations | Every fork is a long-lived maintenance bill; upgrades stall |

## What Second Brain is

A single platform, rolled out to every company, that provides:

1. **Cited Q&A that is correct about permissions.** Answers cite `doc@version` and are built only from content the asker can open *right now*, checked live at the source.
2. **A small set of approval-gated agent actions.** Agents draft; humans send or approve; nothing is "done" without a receipt from the target system.
3. **Isolation by construction.** Each company gets its own cell (database, index, vault namespace, keys, region), stamped from one template.
4. **Self-serve onboarding.** A company's own IT team goes live from one manifest and three commands.
5. **Evidence by default.** An append-only, hash-chained audit log, signed reproducible builds, and evals that gate every release.
6. **A factory, with hand-off.** One factory produces and runs every company's Second Brain, and can later hand custody of one to its company, for example into the company's own cloud account or at divestiture. v1 builds the seams; the hand-off ships in W7 (ADR-0015).

## Goals and measures

Targets are proposed and need parent-org sign-off (see [open questions](../living/OPEN_QUESTIONS.md)).

| Goal | Measure | Proposed v1 target |
|---|---|---|
| Fast time to value | Days from contract to first cited answer | 14 days or less |
| Real adoption | Weekly active users ÷ seats | 40% or more by day 90 |
| Trust | Approve-to-reject ratio on agent drafts | Tracked; reviewed at day 45 |
| Soundness | Law violations in production | 0. Any violation is Sev-1 |
| Self-serve | Onboarding steps needing the platform team | 0 |
| Freshness | p95 time from source change to answerable | 15 minutes or less |
| Latency | p95 time to a cited answer | 6 seconds or less |

"Hours saved" is not a v1 metric. We can't measure it honestly yet, so we don't report it.

## Non-goals for v1

- On-prem or self-hosted installs, self-hosted models, or fine-tuning.
- Kubernetes.
- Autonomous or multi-step write agents, and ERP or CRM writes.
- Per-company custom agents, code paths or forks.
- Executing a custody hand-off. It's planned for W7, and v1 only builds the seams that make it possible (ADR-0015).
- Anything cross-company, including portfolio-wide insights for the parent. If the parent needs to read company data, it becomes a separate tenant with explicit grants and its own audit, never a superuser (ADR-0014, proposed).

## How we'll know we're wrong

We'll treat any of these as a signal to revisit the design, not to work around it:

- Live ACL checks make p95 latency exceed 6 seconds and batching doesn't fix it.
- More than one company in the first three needs a code change, not a manifest change, to go live.
- Approvers reject more than half of drafts after day 45.
- The `unknown` permission rate from a connector stays above 1%. Refusing that often makes the product feel broken, and the fix belongs in the connector.

## See also

[Principles](01-principles.md) · [Architecture overview](architecture/00-overview.md) · [PRD](03-prd.md) · [`sb-laws(7)`](../manual/sb-laws.7.md)
