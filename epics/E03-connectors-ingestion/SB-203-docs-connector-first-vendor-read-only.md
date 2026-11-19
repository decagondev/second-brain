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
| Status | Open |

## Why

Documents are where most tribal knowledge lives.

## Acceptance criteria

- [ ] Passes the contract kit
- [ ] ACLs stored as prefilter; `ACL` call asks the source live
- [ ] Read-only scopes only

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Adapter behind the port
- [ ] ACL mapping (users, groups, links)
- [ ] Contract kit run
- [ ] Rate-limit handling
- [ ] Write or extend the law tests: LAW-01, LAW-13
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
