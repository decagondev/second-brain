---
title: "LAW-17 runbook"
summary: "What to do when the LAW-17 alert fires: hand-off leaves no unlisted access."
---

# LAW-17 runbook: Hand-off leaves no unlisted access

**Alert:** `sb_unlisted_access_paths` is above 0: `sb factory access` found a factory access path to a tenant-custody cell that isn't in its hand-off record. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Sev-1: tell the tenant owner and the tenant's security lead at once. It is their cell.
2. Run `sb factory access TENANT` and record each unlisted path (sb-factory(1)).
3. With the tenant owner's agreement, revoke each path, then run `sb factory access TENANT` again and expect exit 0.
4. Find out how the path appeared, from the audit log and the IaC history, and whether it was used.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-17](../../manual/sb-laws.7.md#law-17).
