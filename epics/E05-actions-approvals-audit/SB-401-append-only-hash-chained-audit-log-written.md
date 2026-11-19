---
title: "SB-401: Append-only, hash-chained audit log written before execution"
summary: "If the audit row isn't there first, the action didn't happen through us."
---

# SB-401: Append-only, hash-chained audit log written before execution

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.1 Audit and kill |
| Stories | US-05.4 |
| Manual | [`sb-audit(7)`](../../manual/sb-audit.7.md), [`sb-audit(5)`](../../manual/sb-audit.5.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-11](../../manual/sb-laws.7.md#law-11) |
| Depends on | [SB-108](../E01-tenancy-identity/SB-108-tenant-scoped-request-context.md) |
| Status | In review |

## Why

If the audit row isn't there first, the action didn't happen through us.

## Acceptance criteria

- [ ] Audit row committed before any tool call executes (LAW-04 replay test) (SB-405)
- [x] Each row includes the previous row hash; `UPDATE` and `DELETE` are revoked at the database role
- [x] Tamper test detects any edit
- [ ] Hourly anchors are signed with a per-cell key, not a factory-wide key; `kind: custody` is a valid audit kind (per-cell Ed25519 signing and `custody` built; the hourly WORM schedule comes with the cell runtime)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Audit table and role grants
- [x] Chain hashing
- [ ] Write-ahead hook in tool bus (SB-405)
- [x] Replay and tamper tests (tamper here; replay with SB-405)
- [x] Write or extend the law tests: LAW-04, LAW-11 (LAW-11 here; LAW-04 with SB-405)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.1a, with SB-402): `internal/core/audit` defines rows, their RFC 8785 canonical form (unsafe strings are refused at write time), `hash = SHA-256(prev ‖ canonical)`, and `Log.Write`, which returns only after the store commits. `Log` is also a `policy.Recorder`, so kernel decisions become `decision` rows. Anchors are Ed25519, from a 32-byte seed the cell unwraps with its own root key. `internal/adapters/postgres/audit` keeps `audit_TENANT.rows` with triggers that refuse UPDATE, DELETE and TRUNCATE for every role, and `GrantWriter` gives the writer role exactly INSERT and SELECT. Appends serialise on a per-tenant advisory lock: 20 concurrent writers give a gap-free chain against real Postgres. There is also a memory store. `TestLAW11_AuditChainUnbroken` went red first, then green; it detects edited, dropped, swapped, duplicated and cut-off rows and forged anchors, and a verifier that only looks for anchors is caught.

**Left:** the tool bus's write-ahead hook and the LAW-04 replay test (SB-405), and the hourly WORM anchor schedule (cell runtime).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
