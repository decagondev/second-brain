---
title: "LAW-02 runbook"
summary: "What to do when the LAW-02 alert fires: no tenant crossing."
---

# LAW-02 runbook: No tenant crossing

**Alert:** `law_violation_total{law="LAW-02"}` rose: the kernel's tenant guard or an index refused another tenant's scope or data. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Treat as a security incident: page security.
2. Kill the affected tenant's cell traffic: `sb kill tenant --tenant T --reason LAW-02`.
3. Export the audit chain (`sb audit export`) and verify it offline before anything else changes.
4. Find the request path from the decision rows; the guard stopped it, so look for why it was attempted.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-02](../../manual/sb-laws.7.md#law-02).
