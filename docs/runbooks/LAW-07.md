---
title: "LAW-07 runbook"
summary: "What to do when the LAW-07 alert fires: no secret in prompt, log or audit."
---

# LAW-07 runbook: No secret in prompt, log or audit

**Alert:** `sb_secret_findings_total` rose: the scanner saw a canary or a resolved secret in a log line, a model request or an audit row. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Rotate the named secret at its source now (the alert names its id, never its value).
2. Find the path from the alert's `where`; purge the affected log segment through the log store's retention controls.
3. Add the path to `tests/canary` so it is scanned in CI forever.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-07](../../manual/sb-laws.7.md#law-07).
