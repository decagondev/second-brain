---
title: "SB-607: Per-tenant observability"
summary: "Health, cost, freshness and errors per tenant."
---

# SB-607: Per-tenant observability

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.3 Operate |
| Stories | US-07.4 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | — |
| Depends on | [SB-208](../E03-connectors-ingestion/SB-208-incremental-sync-via-changes-with-freshness.md), [SB-306](../E04-retrieval-answers/SB-306-ask-agent-and-post-v1-ask.md) |
| Status | Open |

## Why

Health, cost, freshness and errors per tenant.

## Acceptance criteria

- [ ] Dashboards for latency, errors, model cost, freshness, ACL unknown rate
- [ ] No document content in telemetry

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Metrics
- [ ] Dashboards
- [ ] Telemetry redaction test
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
