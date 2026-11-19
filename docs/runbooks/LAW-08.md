---
title: "LAW-08 runbook"
summary: "What to do when the LAW-08 alert fires: at most one behind stable."
---

# LAW-08 runbook: At most one behind stable

**Alert:** `sb_release_lag_max` is above 1: a stable tenant runs a release more than one behind stable. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Run `sb release status` to find the tenant and why it's behind (draining, failed deploy).
2. Redeploy it to current stable; promotion stays blocked by the promote guard until it is current.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-08](../../manual/sb-laws.7.md#law-08).
