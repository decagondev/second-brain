---
title: "SB-306: `ask` agent and `POST /v1/ask`"
summary: "One endpoint that every entry point uses."
---

# SB-306: `ask` agent and `POST /v1/ask`

| Field | Value |
|---|---|
| Epic | [E04 Retrieval and Cited Answers](README.md) |
| Wave / slice | W3 / S3.2 Answers |
| Stories | US-04.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-304](../E04-retrieval-answers/SB-304-answer-composer-with-doc-version-citations.md), [SB-305](../E04-retrieval-answers/SB-305-model-gateway-adapter.md) |
| Status | Open |

## Why

One endpoint that every entry point uses.

## Acceptance criteria

- [ ] Endpoint matches `sb-ask(3)` exactly, including errors
- [ ] Trace id returned and audited

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Handler
- [ ] Request and response types generated from the man page schema
- [ ] Contract tests
- [ ] Write or extend the law tests: LAW-01, LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
