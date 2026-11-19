---
title: "sb-observability(7)"
man: "7"
summary: Metrics, alerts, reports and telemetry rules.
---

# sb-observability(7)

## NAME

sb-observability — see every tenant's health without seeing its content

## SYNOPSIS

```text
metrics: latency · errors · freshness · acl_unknown_ratio · model_cost · law_violation_total
alert:   any law_violation_total > 0  →  Sev-1
```

## DESCRIPTION

### Per-tenant dashboards

Latency (p50, p95), error rates by code, ingest freshness, ACL unknown ratio per connector, refusals by reason, model spend against budget, active users, and agent run outcomes.

### Alerts

| Alert | Condition | Severity |
|---|---|---|
| Any law violation | `law_violation_total{law}` increases | **Sev-1** |
| Attestation mismatch | running digest ≠ signed | Sev-1 |
| Audit chain break | `sb audit verify` fails | Sev-1 |
| Ask latency | p95 > 6 s for 15 min | Sev-2 |
| Freshness | p95 > 15 min | Sev-2 |
| Unknown ratio | > 1% for 30 min on a connector | Sev-3 |
| Budget pace | > 120% | Sev-3 |

Every alert links to a runbook.

### Canaries and the secret scanner

Each cell plants **canary secrets** (SB-411): random values that look like credentials, `sbcanary-<tenant>-<32 hex>`. One is stored in the vault at `vault://tenants/TENANT/canary`, and one is placed in a fixture document in each connected source's test space. They are registered with the log redactor like any secret. Nothing legitimate ever uses them, so seeing one anywhere is proof of a leak path.

The scanner looks for every canary, and for every value resolved from the vault, in three places: each log line as it is written, each model request before it leaves the gateway, and each audit row and export. A finding names the place and the canary's id, **never the value**. It is a **Sev-1** alert (`kind: secret_leak`), sent through the alert webhook. In CI, `tests/canary` runs the ask path, the tool bus and the audit export with canaries planted and fails on any finding. The same test also shows that a path that leaks on purpose is found.

### Weekly leadership report

Weekly active users ÷ seats, asks, cited-answer rate, top uncited topics (as counts, never content), drafts and their approve-to-reject ratio, approved actions and receipts, spend. Delivered automatically to the tenant's approver group.

Exactly (SB-606). The week is Monday 00:00 to Monday 00:00 UTC, and the report goes out on Monday at 07:00 UTC. It is built from the cell's own traces, draft outcomes, actions and metering, and never from content.

| Line | Definition |
|---|---|
| Weekly active users ÷ seats | distinct users who asked or drafted at least once, divided by the directory's active users |
| Asks | answers given (traces) |
| Cited-answer rate | answers with at least one cited claim ÷ answers |
| Gaps | answers by notice, `no_sources`, `no_access` and `most_uncited`, each as a count. These are the places where people asked and the platform had nothing to show them. They are counted by kind, never by question |
| Drafts | `used` and `discarded`, and the ratio of the two |
| Actions | proposed, approved, rejected, expired, `Ran` with a receipt, `DidNotRun` |
| Spend | model tokens and their cost at the release's price table |

The same numbers answer `GET /v1/admin/usage?week=2027-W10` (sb-admin(3)), which operators and approvers may read. The report is Markdown with no links to content, and delivery goes through the cell's notification adapter (email or chat).

### Telemetry rules

Ids only: no content, prompts, answers or secrets in metrics, logs or traces. A CI redaction test and the production secret scanner enforce this (LAW-07).

## LAWS

[LAW-07](sb-laws.7.md#law-07), all laws via Sev-1 alerts.

## EXAMPLES

```text
law_violation_total{tenant="acme-logistics",law="LAW-01"} 0
acl_unknown_ratio{tenant="acme-logistics",connector="chat"} 0.004
```

## SEE ALSO

[sb-admin(3)](sb-admin.3.md), [15 Observability, backup and DR](../docs/architecture/15-observability-backup-dr.md)
