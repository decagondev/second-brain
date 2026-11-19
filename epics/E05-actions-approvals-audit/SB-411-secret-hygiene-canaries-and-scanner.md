---
title: "SB-411: Secret hygiene: canaries and scanner"
summary: "Assume a leak will be attempted; prove it fails."
---

# SB-411: Secret hygiene: canaries and scanner

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.4 Hygiene |
| Stories | US-05.6 |
| Manual | [`sb-audit(5)`](../../manual/sb-audit.5.md) |
| Laws | [LAW-07](../../manual/sb-laws.7.md#law-07) |
| Depends on | [SB-205](../E03-connectors-ingestion/SB-205-connector-credentials-via-vault-references.md) |
| Status | In review |

## Why

Assume a leak will be attempted; prove it fails.

## Acceptance criteria

- [ ] Canary secrets planted in vault and sources never appear in prompts, logs or audit (scanner in CI and prod) (CI: `tests/canary` in `make smoke`; production wiring with the cell runtime)
- [x] Scanner finding is a Sev-1 alert

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Canary generator
- [x] Log and prompt scanner
- [x] Alert wiring
- [x] LAW-07 green
- [x] Write or extend the law tests: LAW-07
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.4a): `internal/core/canary` makes canaries (`sbcanary-<tenant>-<32 hex>`, id `cny_…`). The `Scanner` knows every canary and every resolved secret, so it is a `secrets.Registrar` too. `Watch` turns each finding into a Sev-1 `secret_leak` alert that carries ids only, and `Writer` scans a log stream as it is written. `internal/adapters/alert/webhook` sends alerts over HTTPS only, with one retry on 5xx. The audit log now scrubs every field through the cell's redactor before hashing, and the gateway's `OnSecret` hook fires on a refused prompt. `vault://tenants/T/canary` is a valid reference. `tests/canary` (run by `make smoke`) plants a vault canary and a source canary, resolves and logs one, sends a document holding the other through the gateway, writes an audit row quoting a canary, exports the chain, and scans logs, model request bodies and the export: zero findings and zero alerts, with proof that redaction happened. A control path that leaks on purpose is found by id and line, and alerted without the value. Building it showed that the API key legitimately reaches its own provider in a header, so the scanner reads prompts as request bodies, not headers. `TestLAW07_NoSecretInAudit` went red first, then green, and a log without redaction is caught.

**Left:** wiring the scanner into a running cell's log stream and gateway, and canaries in real source sandboxes (cell runtime, connector sandboxes).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
