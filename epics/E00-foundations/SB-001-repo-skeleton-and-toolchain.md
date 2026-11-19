---
title: "SB-001: Repo skeleton and toolchain"
summary: "Everything later assumes one command builds, tests and checks the repo."
---

# SB-001: Repo skeleton and toolchain

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.1 Skeleton |
| Stories | US-00.5 |
| Manual | [`sb(1)`](../../manual/sb.1.md) |
| Laws | — |
| Depends on | — |
| Status | Done |

## Why

Everything later assumes one command builds, tests and checks the repo.

## Acceptance criteria

- [x] `make check` runs build, unit tests, law tests, lint and docs build from a clean clone
- [x] Go version, Python version and tool versions pinned in one place (`tools/versions.env`)
- [x] CI runs `make check` on every PR

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Create Go workspace `go.work` with `cmd/` and `internal/` per docs/06-engineering-standards.md
- [x] Add `Makefile` targets: `build`, `test`, `laws`, `proofs`, `evals`, `docs`, `check`
- [x] Pin toolchain versions in `tools/versions.env`; CI reads the same file
- [x] Add `golangci-lint` config with the rules in the engineering standards
- [x] Write devlog entry
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `go.mod` (one module, standard library only, ADR-0016), `cmd/sb`, `internal/cli`, `internal/core/ports`, `internal/version`, `.golangci.yml`, the Makefile and `tools/versions.env`. `make check` passes on a clean copy of the repo. CI runs `make check`.

Deviation: a single `go.mod` instead of a `go.work` workspace, since there's only one module (P3). The engineering standards are updated to match.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
