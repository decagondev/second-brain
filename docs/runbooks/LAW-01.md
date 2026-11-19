---
title: "LAW-01 runbook"
summary: "What to do when the LAW-01 alert fires: cite only what the asker can open."
---

# LAW-01 runbook: Cite only what the asker can open

**Alert:** `law_violation_total{law="LAW-01"}` rose: `sb tenant verify` or the live-check guard saw a citation the asker can't open now. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Kill the `ask` agent for the tenant: `sb kill agent ask --tenant T --reason LAW-01`.
2. Find the trace ids in the alert's window (`GET /v1/admin/audit?kind=decision`) and the connector involved.
3. Re-run `sb tenant verify` with `--law LAW-01` to confirm, then check that connector's ACL mapping against the source.
4. Fix, add a regression case to the tenant's evals, then `sb unkill`.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-01](../../manual/sb-laws.7.md#law-01).
