---
title: "SB-409: `draft-reply` agent where a human always sends"
summary: "The first agent saves time and changes nothing on its own."
---

# SB-409: `draft-reply` agent where a human always sends

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.3 Execution |
| Stories | US-05.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md), [SB-405](../E05-actions-approvals-audit/SB-405-tool-bus-policy-audit-then-execute.md) |
| Status | Open |

## Why

The first agent saves time and changes nothing on its own.

## Acceptance criteria

- [ ] Drafts cite sources like answers do
- [ ] Send is a human click in the target app or our UI
- [ ] Approve and reject rates recorded

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Agent prompt and tools in release bundle
- [ ] Draft UI
- [ ] Metrics
- [ ] Write or extend the law tests: LAW-04, LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
