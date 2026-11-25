---
title: "SB-4803: Project kickoff"
summary: "Each project starts from a blank page, and nobody finds what similar projects decided."
---

# SB-4803: Project kickoff

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.1 |
| Stories | US-48.1 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-24, LAW-29 |
| Depends on | SB-4801, SB-4304, SB-4101 |
| Status | Done |

## Why

Each project starts from a blank page, and nobody finds what similar projects decided.

## Acceptance criteria

- [x] **Start a project** creates a collection, a goal draft and a decision log, and drafts a kickoff brief of up to 5 similar past projects
- [x] The kickoff brief cites only what the lead can open
- [x] A member from another space is added only after their steward approves, and sees only what they could already open

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Start a project
- [x] Kickoff brief agent and eval suite
- [x] Cross-space members
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `POST /v1/spaces/NAME/projects` makes the collection (marked with its project), the lead's goal draft, the decision log, a request to the home space's stewards for each member from another space (`POST /v1/projects/ID/members` approves), and starts `kickoff`, whose day-0 step is the kickoff brief: up to 5 closed projects sharing name words, their decisions and published lessons, each line only if the lead can open its cite. The brief is put together by code from records, not by a model, so no new agent definition or eval suite was needed (sb-agents(7), Rhythm drafts); glossary matching was left out, names only.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
