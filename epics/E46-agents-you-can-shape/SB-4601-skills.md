---
title: "SB-4601: Skills"
summary: "A department can't tell its agent how it likes its updates and briefs written."
---

# SB-4601: Skills

| Field | Value |
|---|---|
| Epic | [E46 Agents You Can Shape](README.md) |
| Wave / slice | W45 / S45.1 |
| Stories | US-46.1 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-04, LAW-06, LAW-09 |
| Depends on | SB-4403, SB-3902 |
| Status | Open |

## Why

A department can't tell its agent how it likes its updates and briefs written.

## Acceptance criteria

- [ ] A space names up to 10 skills (name, instruction of at most 1,000 characters, at least 3 eval cases) through a manifest diff
- [ ] `sb agents lint` refuses a skill that names a tool, widens a scope or acts on document text, naming the line
- [ ] `/name` loads it on an ask, research or routine; skills flow down the tree and the MCP server lists them

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest
- [ ] Lint
- [ ] Loader
- [ ] Eval cases
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
