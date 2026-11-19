---
title: "SB-610: CRM connector (read-only)"
summary: "Customer context completes the picture for most teams."
---

# SB-610: CRM connector (read-only)

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W6 / S6.3 Operate |
| Stories | US-03.5 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md) |
| Status | In review |

## Why

Customer context completes the picture for most teams.

## Acceptance criteria

- [x] Passes the contract kit
- [x] Record-level permissions map to ACL

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Adapter
- [x] ACL mapping
- [x] Contract kit run
- [x] Write or extend the law tests: LAW-01
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.3b): `internal/adapters/connectors/hubspot` is `crm: hubspot`. It indexes companies and deals and never contacts. Access is checked live per record. The connector reads the user's role and super admin flag from the settings users API, and their owner id and teams from the owners API. The tenant's role map in the vault says what each role may view: `all`, `team` or `owned`. A role missing from the map is `Unknown`. A user with no account, or a deactivated owner, is `Deny`, and an archived or missing record is `Unknown`. Changes are polled through the CRM search API by `hs_lastmodifieddate`, archived records are deletes, and a record that was just reassigned is reported as a permission-only change. The connector passes the contract kit against `hubspottest`, a recorded fake. A new LAW-01 law test, `TestLAW01_CRMRecordPermissionsAreLive`, checks every live verdict against a reference model of HubSpot's permissions through reassignments, role and team changes, deactivation and archiving, and a self-test shows that reading a team role as view-all is caught. LAW-03 runs over the connector too. The broad token `hubspot:users` is added to the asker's prefilter tokens.

**Left:** a run against a real HubSpot sandbox portal, with a private app the user creates.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
