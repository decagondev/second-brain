---
title: "SB-4403: Agent lint in the gate"
summary: "A bad agent definition is caught only if a reviewer happens to notice."
---

# SB-4403: Agent lint in the gate

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.1 |
| Stories | US-44.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-04, LAW-06, LAW-09 |
| Depends on | SB-4402 |
| Status | Done |

## Why

A bad agent definition is caught only if a reviewer happens to notice.

## Acceptance criteria

- [x] `sb agents lint` runs in `make check` and the release gate
- [x] It fails an undeclared tool, a write outside approvals, no department or template, a restricted space without `restricted: true`, no eval suite, or a prompt that acts on document text, naming the line
- [x] The selftest breaks each rule once and goes red; every shipped agent passes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Lint
- [x] CI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.LintAgents and `sb agents lint` (owner, tool, write, department, restricted, suite, prompt, injection, and a definition for every agent the code runs), each naming its file and line; it runs in `make evals`, so in make check and the release gate. TestAgentLintCatchesEachRule breaks each rule once; every shipped agent passes.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
