---
title: "SB-003: Living documents bootstrap"
summary: "The devlog, bugs log and assurance ledger only work if they exist from the first commit."
---

# SB-003: Living documents bootstrap

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.1 Skeleton |
| Stories | — |
| Manual | — |
| Laws | — |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | Done |

## Why

The devlog, bugs log and assurance ledger only work if they exist from the first commit.

## Acceptance criteria

- [x] `living/` contains DEVLOG, BUGS_AND_MITIGATIONS, ASSURANCE_LEDGER, CHANGELOG, GLOSSARY, OPEN_QUESTIONS and RISKS
- [x] Devlog entry 1 committed
- [x] PR template links the Definition of Done in docs/05-living-documents.md

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Create each file with its header and format note
- [x] Add `.github/pull_request_template.md` with the DoD checklist
- [x] Commit devlog entry 1
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. All seven living documents exist in `living/`, along with the devlog's first entries and `.github/pull_request_template.md`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
