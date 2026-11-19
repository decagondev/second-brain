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
| Status | Done |

## Why

Four calls, versioned, tested by a kit, is how connectors stay boring.

## Acceptance criteria

- [x] Go interface `Connector{Fetch, ACL, Changes, Act}` with API version
- [x] Contract kit any connector can run
- [x] A mock connector passes the kit

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Write the interface and types
- [x] Write the contract kit as a Go test suite
- [x] Version negotiation
- [x] Docs examples
- [x] Write or extend the law tests: LAW-01, LAW-03, LAW-05
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: done.
- `ports.Connector` gains `APIVersion()`, and `ports.Negotiate` refuses any other contract version with `ENOTSUP`.
- `internal/core/connectorkit.Run(t, sandbox)` runs the ten kit cases from 03-connectors: version, version-stable fetch, deny-is-deny, error-is-unknown (500, timeout, 404), a batch of 200 within 800 ms, cursor resume, ACL delta, receipt-or-error, idempotent `Act`, and `ENOTSUP` for read-only connectors.
- `internal/adapters/connectors/mock` is a controllable source plus connector. It passes the kit in read-only and writable modes and is the base for SB-202.
- Connector-level law tests for LAW-03 and LAW-05 were red first (`not implemented`), then green. Self-tests catch allow-on-fault and a non-idempotent `Act`.

The LAW-05 test first passed against nothing, because it swallowed the pending error; the ratchet flagged it, and it was fixed before any code existed. LAW-01 at the connector level is the ACL truth check inside the LAW-03 test (no fault means exactly the source's answer). LAW-01 itself is retrieval-level (SB-302).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
