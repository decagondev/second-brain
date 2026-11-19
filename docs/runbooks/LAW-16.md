---
title: "LAW-16 runbook"
summary: "What to do when the LAW-16 alert fires: the manifest is the truth."
---

# LAW-16 runbook: The manifest is the truth

**Alert:** `sb_drift_findings_total` rose: live config differs from the applied manifest. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Read the drift alert: it names the resource and field.
2. If the change was intended, change the manifest first; otherwise `sb tenant apply` makes the cell match.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-16](../../manual/sb-laws.7.md#law-16).
