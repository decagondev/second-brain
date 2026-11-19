---
title: "LAW-18 runbook"
summary: "What to do when the LAW-18 alert fires: the root key always has a custodian."
---

# LAW-18 runbook: The root key always has a custodian

**Alert:** `sb_unwrappable_keys` is above 0: a data key can't be unwrapped by the cell's root key. Severity: Sev-1 ([sb-observability(7)](../../manual/sb-observability.7.md#law-alerts-exactly-sb-608)).

## Steps

1. Sev-1: data may become unreadable. Page the key custodian.
2. Check the key-wrap inventory for the cell and the custody record (sb-factory(7)).

## After

Log the incident in `living/BUGS_AND_MITIGATIONS.md` with which test would have caught it, and add that test. Law: [LAW-18](../../manual/sb-laws.7.md#law-18).
