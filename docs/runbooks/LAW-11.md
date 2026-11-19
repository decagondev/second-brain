---
title: "LAW-11 runbook"
summary: "What to do when the LAW-11 alert fires: audit chain unbroken."
---

# LAW-11 runbook: Audit chain unbroken

**Alert:** `sb_audit_verify_failures_total` rose: `sb audit verify` found a break. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Sev-1: preserve evidence, page security. Do not write to the store.
2. Run `sb audit verify --export` on the latest backup copies (SB-412) to find the last good seq.
3. The first bad seq names the row; compare every copy.

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-11](../../manual/sb-laws.7.md#law-11).
