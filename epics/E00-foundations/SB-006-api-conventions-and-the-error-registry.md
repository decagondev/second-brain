---
title: "SB-006: API conventions and the error registry"
summary: "One error vocabulary across CLI, API and UI keeps humans, LLMs and scripts aligned."
---

# SB-006: API conventions and the error registry

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.2 First pages |
| Stories | US-00.1 |
| Manual | [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-errors(7)`](../../manual/sb-errors.7.md) |
| Laws | [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-002](../E00-foundations/SB-002-manual-layout-man-page-template-and-docs.md) |
| Status | Done |

## Why

One error vocabulary across CLI, API and UI keeps humans, LLMs and scripts aligned.

## Acceptance criteria

- [x] Every error has a stable code, an HTTP status, a CLI exit code and a one-line remedy
- [x] `EPERM` is documented as covering deny, unknown and missing source

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Draft `sb-api(3)`: versioning, auth, pagination, idempotency, errors
- [x] Draft `sb-errors(7)` registry table
- [x] Generate `internal/errs/codes.go` from the registry table (single source)
- [x] Write or extend the law tests: LAW-03
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `internal/errs/codes.go` is generated from the registry table in `sb-errors(7)` by `tools/gen/errs.py`; `make gen-check` fails if they drift. `ENOTIMPL` was added to the registry first (manual first). Unit tests cover exit codes, HTTP statuses, unknown codes, and the rule that an uncoded error exits 1.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
