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
| Status | Open |

## Why

Code that is hard to read is where the next bug goes unnoticed.

## Acceptance criteria

- [ ] Stricter `golangci-lint` rules for unused code, duplication and long functions, with the biggest files tidied
- [ ] The new linters pass in `make check`; any exception is narrow and says why
- [ ] The tool bus, the manifest checker and the CLI are split or simplified with no change in behaviour (every law test and smoke test unchanged and green)

## Subtasks

- [ ] Update the man pages the change touches first (manual first, P2)
- [ ] Lint rules
- [ ] Tidy the biggest files
- [ ] No behaviour change
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
