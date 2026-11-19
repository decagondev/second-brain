---
title: "ADR-0020: Department brains are spaces inside the company's cell, and a space can only narrow access"
summary: "Department brains are spaces inside the tenant's cell; a space can only narrow what the source allows."
---

# ADR-0020: Department brains are spaces inside the company's cell, and a space can only narrow access

| Status | Date | Deciders |
|---|---|---|
| Accepted (plan, W8 to W13) | 2026-11-19 | Platform lead (the user) |

## Context

Companies want more than one brain: an Engineering brain, a People brain, a Finance brain, and so on, each with its own sources, agents and people. Real companies organise by function (or by division, or as a matrix of both), and their access rules follow that shape. HR, Legal and Finance hold need-to-know data; Engineering and Product share widely; deal teams sit behind information barriers. Today a cell is one undivided brain whose only access rule is the source's ACL. That is sound (ADR-0002), but it can't say "this answer should come only from People sources" or "Finance content never reaches a company-wide answer for someone outside Finance".

The full research and the concept are in [08-department-brains.md](../08-department-brains.md).

## Decision

- A **department brain is a space** inside the company's cell. A space has its own corpus (a scope over the company's connectors), agents, stewards, retention, label default and data key. Every space in a company shares one IdP, one release, one audit chain and one cell. **The cell stays the tenant boundary** (ADR-0001); nothing about LAW-02 changes.
- **A space can only narrow access.** At answer time a chunk is used only if the source's live ACL allows it **and** the space policy allows it. A space can hide what someone could open, never show what they can't (a new law, LAW-19).
- Access is decided by four layers, all in Cedar (ADR-0008): **membership** from IdP groups, **attributes** from SCIM's enterprise extension (department, title level, location, cost centre, manager), a **label** on every document (`public`, `internal`, `confidential`, `restricted`), and **information barriers** between named groups.
- Spaces are declared in the manifest (LAW-16). A space is a manifest diff, proposed by its stewards and approved by the tenant operator.
- **Restricted spaces never feed company-wide answers for non-members** (LAW-20). A company-wide ask is the union of the spaces the asker belongs to, minus restricted content from spaces they don't.
- Each space's data key is wrapped by the cell root key, so a space can be crypto-shredded without touching the rest of the cell, and LAW-18 still holds.

## Alternatives rejected

| Option | Why not |
|---|---|
| A cell per department | Treats two departments of one company like two companies: double cost, no company-wide answers, and a cross-space feature would need the ADR-0014 machinery for colleagues |
| Spaces that can grant access the source denies | Breaks ADR-0002 and LAW-01; stale and hard to audit |
| Department tags with no policy behind them | A filter in retrieval is a convention, not a control; one bug leaks HR content |
| Separate indexes per space with no shared retrieval | Makes the company-wide ask a fan-out across stores with no single place to enforce LAW-20 |

## Consequences

- Access stays explainable: "the source allowed it, and your space allowed it".
- Policy gets bigger. The Alloy model (`Spaces.als`) and policy analysis must prove narrowing for every space policy, not just the base set.
- A department that needs harder separation than a space gives (for example a legally separate subsidiary) is still a separate tenant.
- Supersedes nothing. ADR-0014 still governs any parent-org access; W13 extends it to per-space grants.
