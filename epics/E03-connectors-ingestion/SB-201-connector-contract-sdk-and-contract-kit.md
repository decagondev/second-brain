---
title: "SB-201: Connector contract, SDK and contract kit"
summary: "Four calls, versioned, tested by a kit, is how connectors stay boring."
---

# SB-201: Connector contract, SDK and contract kit

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.1 Contract |
| Stories | US-03.1, US-03.5 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-03](../../manual/sb-laws.7.md#law-03), [LAW-05](../../manual/sb-laws.7.md#law-05) |
| Depends on | [SB-005](../E00-foundations/SB-005-first-drafts-sb-connector-3-sb-permissions-7.md), [SB-109](../E02-policy/SB-109-policy-engine-v0-authorise-returns-allow.md) |
| Status | Open |

## Why

Four calls, versioned, tested by a kit, is how connectors stay boring.

## Acceptance criteria

- [ ] Go interface `Connector{Fetch, ACL, Changes, Act}` with API version
- [ ] Contract kit any connector can run
- [ ] A mock connector passes the kit

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Write the interface and types
- [ ] Write the contract kit as a Go test suite
- [ ] Version negotiation
- [ ] Docs examples
- [ ] Write or extend the law tests: LAW-01, LAW-03, LAW-05
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
