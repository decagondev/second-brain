---
title: "SB-009: Link check, diagram check and manual example extraction"
summary: "Docs that lie are worse than no docs."
---

# SB-009: Link check, diagram check and manual example extraction

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.3 Rules in CI |
| Stories | US-00.5 |
| Manual | [`sb(1)`](../../manual/sb.1.md) |
| Laws | — |
| Depends on | [SB-002](../E00-foundations/SB-002-manual-layout-man-page-template-and-docs.md) |
| Status | Done |

## Why

Docs that lie are worse than no docs.

## Acceptance criteria

- [x] Every internal link and anchor in `site/` resolves
- [x] Every Mermaid block renders without a syntax error
- [x] Manual code blocks tagged `run` are extracted into tests and executed

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Add `tools/check/links.py` to CI
- [x] Add headless Mermaid render check
- [x] Add extractor for ```yaml run``` and ```sh run``` blocks
- [x] Fail CI on any broken link or diagram
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `tools/check/links.py` (217 pages, 0 broken), `tools/check/diagrams.py` (51 diagrams, 0 failures; Mermaid pinned in `tools/check/package.json`), and `tools/check/examples.py` (man-page sections, `run` blocks, example YAML). Mutation-checked with a page whose sections were out of order and whose `run` block didn't parse.

Pending by ticket: `sb tenant check` on manifest examples and `invalid.expected`, both with SB-101.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
