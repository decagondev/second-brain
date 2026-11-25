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
| 16 | [Trust each answer, capture conversations, work where people are](16-trust-capture-reach.md) | Twenty features drawn from other knowledge tools: verified answers, meetings and chat captured, MCP and adoption (W35 to W37) |
| 17 | [One org, many levels; an agent for every department; run the company](17-org-agents-os.md) | Twenty features toward a company OS: spaces as a tree, an agent for every department, goals, updates, policies, requests and budgets (W38 to W40) |
| 18 | [Agents that do the legwork; work that comes to you; govern every agent](18-legwork-inbox-govern.md) | Twenty features from a deeper survey of company OS and second brain tools: research, meeting prep, action items and follow-ups; one inbox, routines and memory; owners, evals and a lint for every agent (W41 to W43) |
| 19 | [Departments work together; agents you can shape; every agent accountable](19-departments-shape-account.md) | Twenty features toward a company OS of department agents: consult and hand off between departments as the asker, the leader's week, skills and redrafts, and value, pauses, caps, shadow runs and an AI mark for every agent (W44 to W46) |
| 20 | [Ways of working; agents that check with you; run AI like a business](20-rhythms-check-run.md) | Twenty features toward a company OS that runs the company's recurring moments: rhythms for onboarding, kickoffs, planning, reorgs and 1:1s, agents that ask, take section comments and run long research with checkpoints, and a command centre, routing policies, personal limits, agent goals and outcome checks (W47 to W49) |
| 21 | [Guard what goes in; write it down together; agents that act carefully](21-guard-write-act.md) | Twenty features: suspect passages skipped, sensitive data masked on every way out, agent history and feature rollouts; suggested edits to sources, drafts edited together and cites to the page or slide; agents started on work events, access lent for one run and drafted next actions (W50 to W52) |

## Architecture

Start with the [overview](architecture/00-overview.md), then the subsystem you're working on. Each subsystem document lists its components, interfaces, failure modes, laws and tickets.

## Decisions

[ADR index](adr/README.md). Twenty-one decisions so far; ADR-0014 is still Proposed pending Q-01. ADR-0020 (department spaces) and ADR-0021 (the GCP runtime) are accepted as the plan for W8 to W13.
