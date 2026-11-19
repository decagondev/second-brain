---
title: "SB-1103: Code connector (read-only)"
summary: "Engineering knowledge lives in repositories, pull requests and issues."
---

# SB-1103: Code connector (read-only)

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.2 Engineering sources |
| Stories | US-11.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-03](../../manual/sb-laws.7.md#law-03), [LAW-07](../../manual/sb-laws.7.md#law-07) |
| Depends on | [SB-201](../E03-connectors-ingestion/SB-201-connector-contract-sdk-and-contract-kit.md) |
| Status | Open |

## Why

Engineering knowledge lives in repositories, pull requests and issues.

## Acceptance criteria

- [ ] GitHub read-only: READMEs, docs folders, issues and pull-request descriptions; source code only if the space opts in
- [ ] ACL from repository permissions and teams; a private repo with unknown access is `Unknown`
- [ ] Secret scanner runs on every chunk; a hit is dropped and counted, never stored (LAW-07)
- [ ] Passes the contract kit against a recorded fake; a real org stays In review

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Connector
- [ ] ACL mapping
- [ ] Secret drop
- [ ] Contract kit
- [ ] Write or extend the law tests: LAW-01, LAW-03, LAW-07
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
