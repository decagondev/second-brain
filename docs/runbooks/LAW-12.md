---
title: "LAW-12 runbook"
summary: "What to do when the LAW-12 alert fires: deprovision revokes fast."
---

# LAW-12 runbook: Deprovision revokes fast

**Alert:** `sb_deactivated_requests_total` rose: a request arrived from a user the directory says is deactivated. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Check SCIM delivery from the IdP and the session revocation path.
2. Revoke the user's sessions now (directory deactivate is idempotent).

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-12](../../manual/sb-laws.7.md#law-12).
