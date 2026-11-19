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
| Status | Done |

## Why

A connector never holds a secret in config, memory dumps or logs.

## Acceptance criteria

- [x] Manifest and config hold only `vault://` references
- [x] Secrets resolved at call time and never logged

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Vault adapter behind `Secrets` port
- [x] Redacting logger
- [x] Canary secret test
- [x] Write or extend the law tests: LAW-07
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: done.
- `internal/core/secrets` accepts only the three documented reference forms, refuses other tenants' paths, refuses to resolve without a redactor, and registers every value before returning it. Errors name the reference, never the value.
- `internal/obs` is the structured logger. It renders every value to text and scrubs registered secrets from the message, attributes, errors, nested groups and `With` attributes, and it redacts secret-named keys.
- `internal/adapters/secrets/localvault` keeps one owner-only file per secret and refuses readable-by-others files.
- The manifest already rejects secret-named fields (SB-101). The OIDC and SCIM secrets are read through functions that the cell wires to the `Resolver`.

LAW-07 has its first law test, red first, then green: canary secrets resolved, then logged by five paths, must never reach the output, and the output must not be empty. Self-tests catch a logger that doesn't know the secrets, and one that writes nothing.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
