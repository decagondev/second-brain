---
title: Design
summary: Problem, principles, domain model, PRD, assurance, standards and threat model.
---

# Design documents

Read top to bottom the first time. After that, the [manual](../manual/README.md) is the reference and these documents explain the why.

| # | Document | What's in it |
|---|---|---|
| 00 | [Problem statement](00-problem-statement.md) | Who has the problem, why existing approaches fail, goals and non-goals |
| 01 | [Principles](01-principles.md) | The eight opinions the platform is built on |
| 02 | [Domain model](02-domain-model.md) | Entities, identifiers, the permission triple, the action lifecycle |
| 03 | [PRD](03-prd.md) | Requirements, personas, epics, stories, waves, slices |
| 04 | [Testing and assurance](04-testing-and-assurance.md) | Tests, laws, models, evals, and what gates what |
| 05 | [Living documents](05-living-documents.md) | Devlog, bugs, ledger formats, and the Definition of Done |
| 06 | [Engineering standards](06-engineering-standards.md) | Layout, dependency rule, style, errors, logging, review |
| 07 | [Threat model](07-threat-model.md) | STRIDE and LLM threats mapped to laws |
| 08 | [Department brains](08-department-brains.md) | How companies organise and control access, department spaces, and running cells on GCP (W8 to W13) |
| 09 | [Team knowledge](09-team-knowledge.md) | What blocks knowledge transfer in a team, and ten small features for it on one sovereign machine (W16) |
| 10 | [Run it, protect it, close the loops](10-run-protect-close.md) | Twenty small features for running a sovereign cell, protecting its data, and W16's gaps (W17 to W19) |
| 11 | [Keep people in the loop, keep knowledge owned, fit the site](11-loop-own-fit.md) | Twenty small features: notifications, owned and right documents, and the site's languages, logs and storage (W20 to W22) |
| 12 | [Easier to ask, easier to steward, easier to operate](12-ask-steward-operate.md) | Twenty small quality-of-life features for askers, stewards and operators (W23 to W25) |
| 13 | [Fresh answers, a tidy corpus, a focused context](13-fresh-tidy-focused.md) | Twenty small features against stale data, document rot and context creep (W26 to W28) |
| 14 | [Show the evidence, fill the gaps, run without surprises](14-evidence-gaps-surprises.md) | Twenty small features: passages and sections on cites, gaps for stewards, early warnings for operators (W29 to W31) |
| 15 | [Ask better, capture knowledge, trust and control](15-ask-capture-trust.md) | Twenty small features: more from each answer, replies kept as notes, people's control of their own data (W32 to W34) |

## Architecture

Start with the [overview](architecture/00-overview.md), then the subsystem you're working on. Each subsystem document lists its components, interfaces, failure modes, laws and tickets.

## Decisions

[ADR index](adr/README.md). Twenty-one decisions so far; ADR-0014 is still Proposed pending Q-01. ADR-0020 (department spaces) and ADR-0021 (the GCP runtime) are accepted as the plan for W8 to W13.
