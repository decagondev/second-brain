---
title: "SB-002: Manual layout, man-page template and docs site in CI"
summary: "The manual is the spec, so it needs a home, a template and a renderer before anything else."
---

# SB-002: Manual layout, man-page template and docs site in CI

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.1 Skeleton |
| Stories | US-00.5 |
| Manual | [`sb(1)`](../../manual/sb.1.md) |
| Laws | — |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | Done |

## Why

The manual is the spec, so it needs a home, a template and a renderer before anything else.

## Acceptance criteria

- [x] `manual/TEMPLATE.md` has NAME, SYNOPSIS, DESCRIPTION, FIELDS/OPTIONS, EXIT STATUS/ERRORS, LAWS, EXAMPLES, SEE ALSO
- [x] `python3 tools/site/build.py` renders every page, with search and light/dark themes
- [x] CI publishes `site/` as a build artefact on every PR

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Write the template with front matter keys `title`, `man`, `summary`
- [x] Wire `make docs` to `tools/site/build.py`
- [x] Add CI job that builds the site and uploads it
- [x] Document the authoring rules in `manual/README.md`
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `manual/TEMPLATE.md`, `manual/README.md` (authoring rules), `tools/site/build.py`. `make docs` builds the site and checks every link. CI uploads `site/` as an artefact on every PR.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
