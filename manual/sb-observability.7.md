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
| A scheduled job missed its run | `sb_job_overdue{job}` is 1 | Sev-1 for `anchor`, `backup`, `access`; Sev-2 otherwise (sb-runtime(7), Schedules) |

Every alert links to a runbook.

### Law alerts, exactly (SB-608)

Every law with a production check has one alert rule, in `internal/core/alerts`, and one runbook, `docs/runbooks/LAW-nn.md`. A test fails the build if a law with a **Watch** line in sb-laws(7) lacks either.

| Law | Signal | Fires when |
|---|---|---|
| LAW-01 | `law_violation_total{law="LAW-01"}` (`sb tenant verify` and the live-check guard) | it increases |
| LAW-02 | `law_violation_total{law="LAW-02"}` (the kernel's tenant guard, index `ETENANT`) | it increases |
| LAW-04 | `sb_audit_write_failures_total` | it increases |
| LAW-06 | `sb_attestation_findings_total` | it increases |
| LAW-07 | `sb_secret_findings_total` | it increases |
| LAW-08 | `sb_release_lag_max` | it is above 1 |
| LAW-11 | `sb_audit_verify_failures_total` | it increases |
| LAW-12 | `sb_deactivated_requests_total` | it increases |
| LAW-16 | `sb_drift_findings_total` | it increases |
| LAW-17 | `sb_unlisted_access_paths` (`sb factory access`, after cutover and monthly) | it is above 0 |
| LAW-18 | `sb_unwrappable_keys` | it is above 0 |

All are **Sev-1**, sent through the alert webhook with the law, the tenant and the runbook path, never content. On GCP the `jobs` runner evaluates them over each scrape of `/metrics` and dispatches them (sb-runtime(7), Endpoints).

### Telemetry labels

Metric label values are ids, codes and names only: they must match `[A-Za-z0-9_.:@/-]` and be at most 128 characters. A value that doesn't match is **replaced** with `invalid` and counted in `sb_telemetry_rejected_total`, so a bug that tries to put text into a label can't leak it (LAW-07).

### Budgets and rate limits (SB-609)

The model gateway checks two limits **before** every model call, and refuses with `EBUDGET` instead of calling:

- **Monthly budget:** the tenant's spend this calendar month (UTC), at the release's price table, plus the call's worst case (its `max_tokens` at the output price), must stay within `budget_usd_month`. Spend is metered from the provider's reported usage after each call.
- **Rate limit:** 30 model calls per user per minute (a token bucket). A refused call is counted in `sb_budget_refusals_total{reason}` with `reason` either `budget` or `rate`.

Nothing degrades to unmetered or unlimited use. If the meter can't be read, the call is refused (`EUNAVAIL`). A budget pace above 120% of the month so far raises the Sev-3 alert above.

### Disaster-recovery game day, exactly (SB-613)

The scenario is **lose the cell's database**, and the target is **RTO 4 hours, RPO 15 minutes, inside the cell's region**. The game day runs quarterly in staging and follows [the runbook](../docs/runbooks/dr-game-day.md). It is scored the same way every time:

| Measure | Definition | Pass |
|---|---|---|
| T0 | when the failure is declared | |
| RTO | from T0 until the restored cell answers its smoke test | ≤ 4 h |
| RPO | from the time of the last audit row in the restored chain until T0 | ≤ 15 min |
| Chain | the restored chain verifies with the cell's pinned key, is row-for-row the backup, and the next row written continues it (LAW-11) | yes |
| Index | the index rebuilt from the sources has the same `(cite, ACL)` set as before the failure (LAW-13) | yes |
| Region | every copy used, and the restored cell, are in the cell's pinned region | yes |

The restore (`internal/core/gameday`) checks every copy as the restore drill does (sb-audit(7)) and restores from the first copy that passes, naming the copies it skipped and why. If two copies pass but hold different bytes for the same night, it refuses and a human chooses. It restores only into empty stores, row by row, refusing any row whose `seq` or `prev` doesn't continue the chain. The approvals snapshot from the same night is restored with it. The index is never restored, only rebuilt (LAW-13).

A game day passes only if every line passes. A failed line is a finding: it goes in the retro and `living/BUGS_AND_MITIGATIONS.md`, and the next game day repeats the scenario. With only the local adapters, backups are nightly, so a rehearsal measures RPO honestly and fails it unless a backup ran within 15 minutes of T0. The 15-minute RPO depends on the cloud adapter's continuous log archive (W7). `tests/gameday` rehearses the whole scenario on local adapters in CI.

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

Ids only: no content, prompts, answers or secrets in metrics, logs or traces. A CI redaction test and the production secret scanner enforce this (LAW-07). The log redactor scrubs each resolved secret as written and in its URL-escaped, base64 (both alphabets, padded or not) and hex forms. An attribute named like a credential (`secret`, `password`, `token`, `key`, `authorization`, `cookie`, `credential`) is redacted whatever its value. A secret mixed with other bytes before it is encoded (HTTP Basic auth) can't be recognised in logs, so callers never log credential headers, and the attribute-name rule is the backstop.

## LAWS

[LAW-07](sb-laws.7.md#law-07), all laws via Sev-1 alerts.

## EXAMPLES

```text
law_violation_total{tenant="acme-logistics",law="LAW-01"} 0
acl_unknown_ratio{tenant="acme-logistics",connector="chat"} 0.004
```

## SEE ALSO

[sb-admin(3)](sb-admin.3.md), [15 Observability, backup and DR](../docs/architecture/15-observability-backup-dr.md)
