---
title: "SB-1403: Clean and concise: stricter lint and tidier big files"
summary: "Code that is hard to read is where the next bug goes unnoticed."
---

# SB-1403: Clean and concise: stricter lint and tidier big files

| Field | Value |
|---|---|
| Epic | [E14 Hardening](README.md) |
| Wave / slice | W13 / S13.3 |
| Stories | US-14.1 |
| Manual | as the change needs (manual first, P2) |
| Laws | none (no behaviour change) |
| Depends on | [SB-1402](SB-1402-fuzzing-and-races.md) |
| Status | Done |

## Why

Code that is hard to read is where the next bug goes unnoticed.

## Acceptance criteria

- [x] Stricter `golangci-lint` rules for unused code, duplication and long functions, with the biggest files tidied
- [x] The new linters pass in `make check`; any exception is narrow and says why
- [x] The tool bus, the manifest checker and the CLI are split or simplified with no change in behaviour (every law test and smoke test unchanged and green)

## Subtasks

- [x] Update the man pages the change touches first (manual first, P2)
- [x] Lint rules
- [x] Tidy the biggest files
- [x] No behaviour change
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S13.3): eleven more linters in `.golangci.yml` (docs/06-engineering-standards.md), and their 33 findings fixed: real off-by-one risks (`strings.Index` used as a slice bound), an `append` aliasing a shared slice in a law test, error type assertions that missed wrapped errors (`errors.As` now), response bodies left open in tests, values assigned and never read, a closure result never used, and duplicated KMS encrypt/decrypt code. Eleven deliberate `nil` errors (an ACL that can't ask answers `Unknown`) are annotated with why. Tidied with no behaviour change: the tool bus is three files (calls, decisions, viewing and editing), the verifier runs one method per law instead of one 46-branch function, the API shares one JSON writer and one operator check, and the barrier package lost two pass-through wrappers. The manifest checker and the CLI were already split across files by concern (`spaces.go`, `steward.go`, `patch.go`, `format.go`; `space.go`, `audit.go`) and needed nothing more. Every law test and smoke test is unchanged and green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
