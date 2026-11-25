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
| Status | Done |

## Why

A department can't tell its agent how it likes its updates and briefs written.

## Acceptance criteria

- [x] A space names up to 10 skills (name, instruction of at most 1,000 characters, at least 3 eval cases) through a manifest diff
- [x] `sb agents lint` refuses a skill that names a tool, widens a scope or acts on document text, naming the line
- [x] `/name` loads it on an ask, research or routine; skills flow down the tree and the MCP server lists them

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest
- [x] Lint
- [x] Loader
- [x] Eval cases
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `skills` in a space's manifest (`manifest.Skill`: `instruction` of at most 1,000 printable characters and 3 to 10 `cases`, `"QUESTION": KIND:ID` in the space's scope; at most 10, names `[a-z][a-z0-9-]{0,39}`), written back by `Format`, in the published schema, and merged down the tree by `Effective` (the team's own winning). `manifest.SkillProblem` refuses an instruction that names a tool or an action, widens what is searched or tells the model to act on document text, at the instruction's line; `sb agents lint --manifest FILE` reports the same (`cli.SkillProblems`), and lint's bundle now also requires `catch-up` and `leader-week` definitions. `ask.SkillName` and `Agent.loadSkill` take `/NAME ` off a question before anything else, so search and the live checks see the rest only; the instruction goes after the agent's prompt (`skillLine`), and `Response.Skill` and its trace record it. An unknown skill, one in a company-wide ask or nothing after the name is `EINVAL`. Research plans from the question without the skill and asks each step with it; a routine's saved text carries it. `GET /v1/spaces/NAME/skills` lists a member's (`Stewards.SkillsIn`), and the MCP server answers `prompts/list` and `prompts/get` (`Agent.UsableSkills`). Each skill's cases are `sk_…` cases of the tenant's set (`Cell.withSkillCases`), run as the space's first active steward on every release, outside the 50-case limit and never deleted by hand. Web: `/` in the ask box lists the space's skills (`SkillHint`), and an answer says which skill wrote it. Tests: `manifest/skills_test.go`, `ask/skills_test.go`, `cli/skills_test.go`, `tests/cell/shape_test.go`, `web/src/shape.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
