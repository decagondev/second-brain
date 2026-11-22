---
title: "SB-1707: Handover brief for a successor"
summary: "When someone leaves or changes role, the documents they own and the context around them are the knowledge most often lost."
---

# SB-1707: Handover brief for a successor

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.3 |
| Stories | US-17.3 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-policy(5)`](../../manual/sb-policy.5.md) |
| Laws | LAW-01, LAW-09, LAW-12, LAW-14 |
| Depends on | [SB-1702](SB-1702-who-to-ask.md) |
| Status | Open |

## Why

When someone leaves or changes role, the documents they own and the context around them are the knowledge most often lost.

## Acceptance criteria

- [ ] `handover`, a read-only agent in the release bundle, asked with a person's directory id, briefs the documents that person owns that the asker can open, grouped by location, every line cited, uncited lines dropped
- [ ] It adds the count of that person's documents the asker can't open, never their names
- [ ] Policy decides who may ask: the person's manager, a steward of one of their spaces, or the person; anyone else is `EPERM`
- [ ] An eval suite, `evals/suites/agent-handover.yaml`, in the gate

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Prompt and agent definition
- [ ] Policy rule
- [ ] Eval suite and tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Planned 2026-11-22 with W16, accepted by the user the same day ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
