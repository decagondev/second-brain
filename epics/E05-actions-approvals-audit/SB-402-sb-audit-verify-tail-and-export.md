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
| Status | In review |

## Why

Auditors need tools, not database access.

## Acceptance criteria

- [x] `sb audit verify` checks the full chain and exits non-zero on a break
- [ ] `sb audit export --since` writes signed NDJSON (`audit.Export` from any seq is built; the CLI needs a running cell, and transport encryption comes with its KMS key)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] CLI commands (`verify --export` offline; `tail` and `export` against a running cell come with SB-208)
- [x] Export signing
- [x] Golden tests
- [x] Write or extend the law tests: LAW-11
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S4.1a): `audit.Export` writes a signed start anchor on the row before the first exported row, then the rows, an anchor every 10,000 rows, and one on the last row. `audit.Verify` checks seq continuity, tenant, prev, hash, id and kind, every anchor's signature and match, and the final anchor. With `--key` it pins the cell's key; without it, it reports `key not pinned`. `sb audit verify --export FILE [--key …]` exits 0 or 1 `ECHAIN`, naming the first bad seq. CLI tests cover an intact export, an unpinned key, a one-character tamper and the not-yet-built online form.

**Left:** `sb audit tail` and `sb audit export` against a running cell, and export encryption with the cell's KMS key (cell runtime, SB-208).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
