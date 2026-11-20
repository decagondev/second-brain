---
title: "SB-1105: Accounting connector (read-only, restricted)"
summary: "Finance asks about budgets, actuals and close status; the ledger is restricted."
---

# SB-1105: Accounting connector (read-only, restricted)

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.3 People and Finance sources |
| Stories | US-11.3 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md), `sb-departments(7)` (new) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-07](../../manual/sb-laws.7.md#law-07), LAW-20 (new, W8) |
| Depends on | [SB-1106](SB-1106-field-level-allow-lists-for-restricted-sources.md) |
| Status | In review |

## Why

Finance asks about budgets, actuals and close status; the ledger is restricted.

## Acceptance criteria

- [x] QuickBooks Online read-only: reports, budgets and the chart of accounts
- [x] Bank and card details, tax ids and payroll never ingested
- [x] Passes the contract kit against a recorded fake (a real sandbox company stays In review)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Connector
- [x] Field allow-list
- [x] Contract kit
- [x] Write or extend the law tests: LAW-01, LAW-07, LAW-20
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S11.3c): sb-connector(3) gained `accounting: quickbooks-online`, and `accounting` is a kind of the manifest, kill switches, vault references and asks. `internal/adapters/connectors/quickbooks` reads one company with a token and the tenant's readers from the vault: accounts as structured documents with six declared fields, budgets as text, and the profit and loss and balance sheet for each of the last 12 months. Account and routing numbers, card details, tax ids and payroll are never decoded, declared or requested, so no allow-list can let them in. QuickBooks doesn't expose who may see its reports, so access is the tenant's `readers` list, for documents QuickBooks still holds; a failed read is `Unknown`. Changes come from change data capture; any account change re-reads the reports; a readers change is permission-only for every document. `qbotest` records the API; the connector passes the contract kit. Law tests: `TestLAW03_QuickBooksUnknownIsNeverAllow`, `TestLAW01_QuickBooksReadersAreLive` and `TestLAW07_QuickBooksNumbersNeverStored` (a connector that passes the account number through is caught; every record in the finance space is restricted).

**Left:** a run against a real sandbox company, and refreshing the hourly access token from the refresh token in the cell runtime (SB-903).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
