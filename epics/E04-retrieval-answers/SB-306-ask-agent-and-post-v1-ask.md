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
| Status | In review |

## Why

One endpoint that every entry point uses.

## Acceptance criteria

- [x] Endpoint matches `sb-ask(3)` exactly, including errors
- [ ] Trace id returned and audited (returned, in `X-SB-Trace`, and stored with its decisions; the audit-chain record comes with SB-401)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Handler
- [x] Request and response types checked against the man page's field tables and example (contract test)
- [x] Contract tests
- [x] Write or extend the law tests: LAW-01, LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.3): `internal/core/ask` is the agent: it validates, retrieves and checks live (with a `sources` filter), skips the model when nothing is allowed (`no_access` or `no_sources`), composes through the gateway with the bundled `answer` prompt, logs each stripped label as `citation_spoof`, sets `most_uncited`, and stores a trace with its decision list. `retrieve.Context` now returns that list (`allowed`, `source_denied`, `source_unknown`, `policy_denied`). Conversations send only earlier questions to the model. `internal/api` serves `POST /v1/ask` and `GET /v1/traces/{id}` behind the auth middleware, with the sb-api(3) error shape and `X-SB-Trace`/`X-SB-Release`. A contract test reads sb-ask(3) and checks the request fields and the example response against the Go types. The laws' ask system now runs through this agent with a stand-in model that cites everything and tries a spoofed label: LAW-01 to LAW-03 stay green, and the new `TestLAW14_AskAnswersCitedOrLabelled` is green with its mutation caught. Traces live in an in-memory store for now.

**Left:** the audit-chain record of each trace (SB-401) and a Postgres trace store (with the cell runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
