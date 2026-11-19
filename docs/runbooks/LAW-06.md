---
title: "LAW-06 runbook"
summary: "What to do when the LAW-06 alert fires: only signed, gated builds run."
---

# LAW-06 runbook: Only signed, gated builds run

**Alert:** `sb_attestation_findings_total` rose: a cell attested a digest, release or signature that doesn't match what was deployed. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Treat as a security incident: page security.
2. Kill the tenant: `sb kill tenant --tenant T --reason LAW-06`.
3. Compare the attestation with the bundle (`bundle.json`) and the cell's image; preserve both.
4. Redeploy from a verified release; rotate the cell's attestation key if the cell may be compromised.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-06](../../manual/sb-laws.7.md#law-06).
