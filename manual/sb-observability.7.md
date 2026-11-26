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
| Budget warning | this month's spend passes 80% of `budget_usd_month`, or the month-end forecast passes the budget (SB-3505), once a month (`budget_warning`, SB-1608) | Sev-3, to the tenant operator |
| Box check | a self-hosted machine's check fails: a disk over 90%, a certificate within 7 days, no dump for 50 hours, a service down (`box_check`, `where` the check, `ref` the day; SB-1802) | Sev-2, to the tenant operator, once a day per check |
| An access review is overdue | `sb_access_review_overdue{space}` is 1: a review open 14 days (sb-spaces(7), Access reviews) | Sev-2, to the tenant operator |
| Guardrail flags doubled | a source's injection flags in the last full week are at least 5 and at least twice the week before's (`guardrail_flags_doubled`, `where` the source, `ref` the week; SB-4406) | Sev-3, to the tenant operator, once a week per source |
| A scheduled job missed its run | `sb_job_overdue{job}` is 1 | Sev-1 for `anchor`, `backup`, `access`, `review`; Sev-2 otherwise (sb-runtime(7), Schedules) |

Every alert links to a runbook.

### Health as gauges, exactly (SB-3203)

`/metrics` carries every line of `GET /v1/admin/health` (sb-admin(3)) as two gauges, read each time `/metrics` is scraped: `sb_health_warning{check}` is 1 when the line warns and 0 when it doesn't, and `sb_health_unknown{check}` is 1 when the cell can't read it. A line it can't read is always 1 on `sb_health_unknown`, so it is never 0 on both (LAW-03). The checks are `job:NAME` (overdue; unknown when its record is unreadable), `connector:KIND` (stale), `box:CHECK` (`warn` or `fail`; unknown for `unknown`), `notify`, `index`, `prompt`, `answer_time`, `box:disk runway` and `stewards` (a space with no active steward; unknown when the directory couldn't say), each only when health has that line. When health can't be built at all, `sb_health_unknown{check="health"}` is 1. Labels are check names only, never a person or a document. The existing alerts can use them, for example:

```yaml
- alert: CellHealthWarning
  expr: max by (check) (sb_health_warning) == 1
  for: 30m
- alert: CellHealthUnknown
  expr: max by (check) (sb_health_unknown) == 1
  for: 30m
```

The connectors runbook ([docs/runbooks/connectors.md](../docs/runbooks/connectors.md)) says what to do for each `connector:` line, by its `failure`.

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

### Sharing changes (SB-5403)

A sharing change seen at sync that can't be kept counts `sb_sharing_unrecorded_total{connector}`; the sync itself goes on, because the index already holds the new ACL copy and every read checks it live.

### Telemetry labels

Metric label values are ids, codes and names only: they must match `[A-Za-z0-9_.:@/-]` and be at most 128 characters. A value that doesn't match is **replaced** with `invalid` and counted in `sb_telemetry_rejected_total`, so a bug that tries to put text into a label can't leak it (LAW-07).

### Budgets and rate limits (SB-609)

The model gateway checks two limits **before** every model call, and refuses with `EBUDGET` instead of calling:

- **Monthly budget:** the tenant's spend this calendar month (UTC), at the release's price table, plus the call's worst case (its `max_tokens` at the output price), must stay within `budget_usd_month`. Spend is metered from the provider's reported usage after each call.
- **Rate limit:** 30 model calls per user per minute (a token bucket). A refused call is counted in `sb_budget_refusals_total{reason}` with `reason` either `budget` or `rate`.

Nothing degrades to unmetered or unlimited use. If the meter can't be read, the call is refused (`EUNAVAIL`). A budget pace above 120% of the month so far raises the Sev-3 alert above. **Before the budget runs out** (SB-1608): the first metered call that takes the month's spend past 80% of `budget_usd_month` sends one `budget_warning` alert (Sev-3, `where` `budget`, `ref` the month, e.g. `2027-03`), so operators hear of it while answers still work. It goes once a month, whatever the number of services or restarts: a marker in the usage ledger (`state/usage/once-budget-warning-TENANT-MONTH`) is made with an exclusive create before the alert is sent, so only the first wins; a meter or marker that can't be read sends nothing, and the next metered call tries again. A budget of 0 sends none.

### Spend by space, exactly (SB-4105)

Each model call an ask or a scheduled run makes **in a space** is metered for the tenant and for that space (`space_usd` in the usage ledger: an amount, a month and the space, never who or what). A space's **branch** spend is its own and its teams'. `GET /v1/admin/costs/spaces?month=2027-03` (sb-admin(3)) lists every space: `{ "month", "spaces": [{ "space", "parent"?, "own_usd", "spent_usd", "budget_usd_month", "forecast_usd"? }] }`, `spent_usd` being the branch's, and for the running month after 3 days a straight-line month-end forecast. A space with `budget_usd_month` (sb-space(5)) is watched by the `notify` job: at 80% of it its stewards are told once that month, "The finance space has used 85% of its monthly budget", and at 100% once more, "The finance space has spent its monthly budget: its scheduled briefs and updates pause until next month, and asks carry on". While a branch, or the department above it, is at its budget, its weekly briefs and updates don't run, and its stewards are told why (sb-agents(7)). Asks are never refused for a space's budget: only the tenant's own budget refuses a call (above). Spend that can't be read pauses the runs, never lets them through.

### AI use, exactly (SB-5001)

Every trace carries the `feature` that asked it (`ask`, `research`, `brief`, `routine`, `rhythm`, `consultation`, or `eval`) beside its `tier` and `cost_usd`. `GET /v1/admin/ai/usage` (sb-admin(3)) counts 12 weeks of them by week, space, feature, agent and tier, with each space's budget and spend this month, each capped agent's spend and the personal limit's bands. Eval runs aren't counted. A row of fewer than 5 answers, and a band of fewer than 5 people, shows as "fewer than 5", and no person is named anywhere in it. The CSV has the same numbers.

**Personal limits** (SB-5003). With `limits` in the manifest (sb-tenant(5)), each model call is metered for its person too (`person_usd` in the usage ledger: an amount, a month and the person's id, read only to compare with the limit and to count the bands). At 80% the person is told once that month (notification kind `limit`); at 100% their own asks, consultations and research use the small tier, and their routines pause, until the month turns or a steward lifts it (audited). A ledger that can't be read limits nobody: the tenant's budget still holds.

### Checks and the admin's week (SB-5005, SB-5006)

The numbers a check compares (sb-agents(7), Did it help?) and the admin's week (sb-admin(3)) are counted from the same traces, queues, feedback, audit rows and usage ledger as the pages above, at most once an hour, and keep no text: a check's file in `state/checks` holds its change, its counts and its verdict.

### Cost of one answer (SB-4606)

Every trace keeps `cost_usd`, the model spend of its one ask counted by the gateway at the configured prices, a consultation's within it, and **How this was made** shows it to the person who asked (sb-ask(3)). A report's cost is its own `cost_usd`. It is a count from the gateway, never from the model, and it is the same spend the budgets count (Budgets and rate limits, above).

### Guardrail events (SB-5103)

Each guardrail event is an audit row of kind `guardrail` (sb-audit(5)), written by `sys_guard`, so the audit stream (sb-audit(7), SB-2302) carries each once:

| Event | Subject | State | Space, reason |
|---|---|---|---|
| A suspect passage of a kept answer (sb-ingest(7), Suspect passages) | `suspect:CITE` | `skipped` or `monitored` | the answer's space; reason `agent=NAME` |
| A response, notice or chat reply with something masked on its way out (sb-ingest(7), Nothing sensitive goes out) | `masked:WAY` (`api`, `notify`, `slack`, `mattermost`) | `masked` | reason `strings=N` |

Never the asker, the question or any text. `GET /v1/admin/guardrails/events?since=7d` (sb-admin(3)) lists these with tool calls refused by an approver and denied by the policy kernel, oldest first, at most 10,000: `{ "events": [{ "at", "kind", "space", "agent", "document", "count" }] }`, `kind` one of `suspect`, `masked`, `refused` or `denied`. `since` is `Nd` (1 to 90) or an RFC 3339 time, default 7 days; `format=csv` gives the same columns as CSV. An eval run's answers write no events.

### Guardrail insights, exactly (SB-4406)

`GET /v1/admin/guardrails` (sb-admin(3)) and the **Guardrails** card on the admin console show, for each of the last 12 ISO weeks (UTC):

| Count | From |
|---|---|
| `flags` | chunks the index first saw that week carrying `instruction_like` or `hidden_text_removed` (sb-ingest(7), Content is data), by source kind and space |
| `flagged_uploads` | asks about a file (SB-3802) whose passages carried either flag |
| `refused_tools` | tool calls an approver rejected (`approval` rows, `Rejected`) |
| `denied_tools` | tool calls the policy kernel denied (`decision` rows for `call_tool` or `propose`, not `allow`) |
| `dropped_uncited` | answers with at least one uncited claim dropped (`dropped` above 0) |

Counts only: a count of 1 to 4 is shown as "fewer than 5", and no chunk text, title, question or person is read into it. `doubled` names each source whose flags in the last full week are at least 5 and at least twice the week before's, and the `notify` job raises `guardrail_flags_doubled` for it once that week (Alerts, above).

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

`sb secrets scan [--known PATH]... PATH...` runs the same scanner over files (SB-1808): it learns every value stored under each `--known` path (a vault directory, or a file of one secret), whole and in part (each line of a multi-line value such as a PEM key, and the secret-named fields of a JSON value, such as `secret_key` but not `access_key`), then reads every regular file under each `PATH` and reports each line holding a known value or the shape of a secret (a private key's header, a URL with a password, a bearer token), as `FILE:LINE ID`, never the value. A file over 64 MiB is a finding, not a skip. Any finding exits 1 with `ELAW`; `--json` gives `{files, known, findings}`. A self-hosted box's support bundle is scanned with it before it is written (sb-selfhost(7), Support bundle).

### Weekly leadership report

Weekly active users ÷ seats, asks, cited-answer rate, top uncited topics (as counts, never content), drafts and their approve-to-reject ratio, approved actions and receipts, spend. Delivered automatically to the tenant's approver group.

Exactly (SB-606). The week is Monday 00:00 to Monday 00:00 UTC, and the report goes out on Monday at 07:00 UTC. It is built from the cell's own traces, draft outcomes, actions and metering, and never from content.

| Line | Definition |
|---|---|
| Weekly active users ÷ seats | distinct users who asked or drafted at least once, divided by the directory's active users |
| Asks | answers given (traces) |
| Searches | searches without the model (SB-1701), counted apart from asks |
| Asks refused by the per-person limit | asks refused with `ERATE` for `asks_per_minute` (SB-2305; sb-tenant(5)), counted apart from asks, never by person or question |
| Cited-answer rate | answers with at least one cited claim ÷ answers |
| Gaps | answers by notice, `no_sources`, `no_access` and `most_uncited`, each as a count. These are the places where people asked and the platform had nothing to show them. They are counted by kind, never by question |
| Drafts | `used` and `discarded`, and the ratio of the two |
| Actions | proposed, approved, rejected, expired, `Ran` with a receipt, `DidNotRun` |
| Stale answers | (SB-2807) cited answers that carried `notice_stale` (most of their cites stale, sb-ask(3)) ÷ cited answers, for the cell, as a count and a share; never which documents, titles or questions |
| Spend | model tokens and their cost at the release's price table |
| Input tokens per model call | (SB-2906) the 95th percentile of the input tokens of each model call this week, and last week's, from the usage ledger; marked **growing** when it is over 1.5 times last week's. Counts only |
| Seconds per answer | (SB-3205) the 95th percentile of the seconds each answer took this week, and last week's, from the usage ledger, which records each answer's duration in milliseconds beside its tokens; marked **slower** when it is over 1.5 times last week's. Durations only, never a question |
| Asks on the small model | (SB-4405) asks whose trace records `tier` `small` ÷ asks, as a count and a share, when sb-cell(5) sets `model.small` |
| Notifications | (SB-2607) the notify job's last run: when, the channel, and its counts sent, skipped and failed; never who |
| Usage by space | (SB-3506) one row per space the week's asks were asked in, and `company`: its asks, the share that found nothing (`no_sources`, SB-3101) and the share rated helpful (the answer's latest feedback `up`). A space with fewer than 5 asks shows **fewer than 5** and no shares, so a small space can't be read as one person; the rows of the spaces with 5 or more add up to the totals less those. The steward screen shows its own space's row only, over the last 7 days (sb-ask(3), Documents to review). A team's row follows its department's, as `finance / ap` (SB-3905), and a department's stewards see their teams' counts under **Teams** |
| Adoption | (SB-3804) "31 of 120 seats asked this week (27 last week)": the distinct people who asked at least once (asks only; searches and asks refused by the limit don't count), of the directory's active users, and the same count for the 7 days before. Then, by space, the people who asked in it: "finance 12, ops 9", only for spaces where at least 5 people asked, so a small space can't be read as one person; the rest are left out, never shown as a smaller number. Counts only: no one is named, in the report or in `GET /v1/admin/usage` (`adoption`: `{ "asked", "asked_last_week", "seats", "spaces": { "finance": 12 } }`) |

The same numbers answer `GET /v1/admin/usage?week=2027-W10` (sb-admin(3)), which operators and approvers may read. The report is Markdown with no links to content, and delivery goes through the cell's notification adapter (email or chat). Exactly (SB-1608): the `report` job writes it to `state/reports/WEEK.md` and posts it as JSON, `{ "kind": "weekly_report", "tenant", "week", "markdown" }`, to `alerts.report_webhook`, or to `alerts.webhook` when that is the only one set (sb-cell(5)). A delivery that fails after its retries fails the run, so its missed finish raises `job_missed`; with neither webhook set the file is the delivery.

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
