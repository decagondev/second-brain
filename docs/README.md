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

## Architecture

Start with the [overview](architecture/00-overview.md), then the subsystem you're working on. Each subsystem document lists its components, interfaces, failure modes, laws and tickets.

## Decisions

[ADR index](adr/README.md). Sixteen decisions so far; ADR-0014 is still Proposed pending Q-01, and ADR-0015 (factory and hand-off) is accepted as a direction, with implementation in W7.
