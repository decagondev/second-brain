---
title: "SB-204: Chat connector (read-only)"
summary: "Decisions get made in chat; that knowledge should be findable by those who could see it."
---

# SB-204: Chat connector (read-only)

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.2 First sources |
| Stories | US-03.1 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md), [SB-205](../E03-connectors-ingestion/SB-205-connector-credentials-via-vault-references.md) |
| Status | Open |

## Why

Decisions get made in chat; that knowledge should be findable by those who could see it.

## Acceptance criteria

- [ ] Passes the contract kit
- [ ] Channel membership maps to ACL; private channels respected
- [ ] Direct messages are excluded by default

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Adapter
- [ ] Membership mapping
- [ ] DM exclusion test
- [ ] Contract kit run
- [ ] Write or extend the law tests: LAW-01
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
