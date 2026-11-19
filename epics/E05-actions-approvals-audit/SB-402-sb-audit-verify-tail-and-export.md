---
title: "SB-402: `sb audit` verify, tail and export"
summary: "Auditors need tools, not database access."
---

# SB-402: `sb audit` verify, tail and export

| Field | Value |
|---|---|
| Epic | [E05 Actions, Approvals, Audit and Kill Switches](README.md) |
| Wave / slice | W4 / S4.1 Audit and kill |
| Stories | US-05.4 |
| Manual | [`sb-audit(1)`](../../manual/sb-audit.1.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11) |
| Depends on | [SB-401](../E05-actions-approvals-audit/SB-401-append-only-hash-chained-audit-log-written.md) |
| Status | Open |

## Why

Auditors need tools, not database access.

## Acceptance criteria

- [ ] `sb audit verify` checks the full chain and exits non-zero on a break
- [ ] `sb audit export --since` writes signed NDJSON

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] CLI commands
- [ ] Export signing
- [ ] Golden tests
- [ ] Write or extend the law tests: LAW-11
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
