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
| Status | In review |

## Why

Health, cost, freshness and errors per tenant.

## Acceptance criteria

- [ ] Dashboards for latency, errors, model cost, freshness, ACL unknown rate (`ops/dashboards/tenant.json`; loading it in Grafana against a running cell is W7)
- [x] No document content in telemetry

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Metrics
- [ ] Dashboards
- [ ] Telemetry redaction test
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.3a, with SB-608 and SB-609): `obs.Metrics` gained histograms (`Observe`) and the telemetry rule. A label value that isn't an id, code or name (at most 128 characters of `[A-Za-z0-9_.:@/-]`) is replaced with `invalid` and counted, so text can't reach a label; a test tries a question as a label and finds only `invalid`. `obs.Catalog` lists every metric a cell emits. The ask agent now records `sb_ask_seconds` and `sb_asks_total{outcome}`, and the live check records `sb_acl_verdicts_total{connector,verdict}`. `ops/dashboards/tenant.json` has latency p50 and p95, outcomes, the ACL unknown rate, freshness, tokens, budget refusals and law violations, and a test checks it queries only catalogued metrics.

**Left:** the cell's `/metrics` endpoint and Grafana against a running cell (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
