---
title: "SB-205: Connector credentials via vault references"
summary: "A connector never holds a secret in config, memory dumps or logs."
---

# SB-205: Connector credentials via vault references

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.2 First sources |
| Stories | US-03.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-07](../../manual/sb-laws.7.md#law-07) |
| Depends on | [SB-103](../E01-tenancy-identity/SB-103-cell-template-infrastructure-as-code.md) |
| Status | Open |

## Why

A connector never holds a secret in config, memory dumps or logs.

## Acceptance criteria

- [ ] Manifest and config hold only `vault://` references
- [ ] Secrets resolved at call time and never logged

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Vault adapter behind `Secrets` port
- [ ] Redacting logger
- [ ] Canary secret test
- [ ] Write or extend the law tests: LAW-07
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
