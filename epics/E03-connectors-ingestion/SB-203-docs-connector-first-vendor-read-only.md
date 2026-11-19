---
title: "SB-203: Docs connector (first vendor, read-only)"
summary: "Documents are where most tribal knowledge lives."
---

# SB-203: Docs connector (first vendor, read-only)

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.2 First sources |
| Stories | US-03.1 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md), [SB-205](../E03-connectors-ingestion/SB-205-connector-credentials-via-vault-references.md) |
| Status | In review |

## Why

Documents are where most tribal knowledge lives.

## Acceptance criteria

- [x] Passes the contract kit
- [x] ACLs stored as prefilter; `ACL` call asks the source live
- [x] Read-only scopes only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Adapter behind the port
- [x] ACL mapping (users, groups, links)
- [x] Contract kit run
- [x] Rate-limit handling
- [x] Write or extend the law tests: LAW-01, LAW-13
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: the user chose Google Drive. Built and passing the contract kit against `drivetest`, a recorded fake of Drive v3 plus Google's token endpoint. The fake verifies the JWT assertion's signature, honours domain-wide delegation and enforces per-user visibility.
- `internal/adapters/connectors/gdrive` asks only for `drive.readonly`. The service account key is read from the vault at call time, and access tokens are cached per subject and registered with the log redactor.
- Fetch exports Google Docs as text, downloads `text/*`, and indexes other types by name. The prefilter maps users, `group:`, `domain:` and `anyone` permissions.
- ACL asks Drive as the asker. 403 or 404 is a Deny only if the service account still sees the file; a trashed or missing file is Unknown. It runs 8 at a time within 800 ms and retries rate limits.
- Changes follow Drive's page tokens. A permission-only change is detected by comparing its time with `modifiedTime`, and an empty cursor lists everything (first sync and rebuild).

Law tests: connector-level LAW-03 and LAW-01 truth (`TestLAW03_DriveUnknownIsNeverAllow`) and LAW-13 (`TestLAW13_DriveIndexIsDerived`) run on the Drive connector. They were written after the connector, so there was no red phase; the kit and the existing self-tests cover the checks themselves.

**Left:** a run against a real Google Workspace sandbox, which needs a service account with domain-wide delegation from the user. Rate limits and paging are proven only against the fake until then.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
