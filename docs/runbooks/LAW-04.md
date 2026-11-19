---
title: "LAW-04 runbook"
summary: "What to do when the LAW-04 alert fires: audit before act."
---

# LAW-04 runbook: Audit before act

**Alert:** `sb_audit_write_failures_total` rose: an audit write failed, so the action it guarded didn't run. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Check the audit store's health (database, disk, role grants).
2. Nothing ran unaudited by design; confirm with `sb audit verify`.
3. Once writes succeed, users can retry; no replay is needed.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-04](../../manual/sb-laws.7.md#law-04).
