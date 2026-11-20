---
title: "SB-1107: Per-template eval suites in the gate"
summary: "A template or agent change mustn't reach a tenant unless its department's suite passes."
---

# SB-1107: Per-template eval suites in the gate

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.4 Department evals |
| Stories | US-11.4 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06), LAW-20 (new, W8), LAW-22 (new, W8) |
| Depends on | [SB-1102](SB-1102-department-agents.md), [SB-807](../E00-foundations/SB-807-space-leak-eval-scaffold.md) |
| Status | In review |

## Why

A template or agent change mustn't reach a tenant unless its department's suite passes.

## Acceptance criteria

- [x] Every template ships a citation suite and a space-leak suite
- [x] The release gate (SB-502) blocks on any regression in either
- [ ] A tenant's own space suites run in `sb tenant verify`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Suites
- [x] Gate wiring
- [ ] Verify wiring
- [x] Write or extend the law tests: LAW-06, LAW-20, LAW-22
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S11.4): sb-evals(7) gained "Template suites" and a row for them. Twelve suites, `evals/suites/template-NAME.yaml`, one per catalogue template but `custom`, run over one shared world with a space per template, a member each and an outsider the source wrongly lets open everything: the member cites the space's guide and its sensitive plan in the space, the outsider never cites the plan company-wide (LAW-20), the internal guide still answers the outsider, and for `executive` a barrier keeps even that from the outsider (LAW-22). All 48 cases pass with the extractive model; every suite is in `evals/gate.yaml` with `failures_max: 0` and runs in `make evals`. The barrier case found a bug: barriers matched groups only when both sides were written `group:NAME`, so a world, or a manifest mixing forms, could miss one; `spaces.ReadContext` now compares names, with a test. The new law test `TestLAW06_EveryTemplateSuiteGatesTheRelease` checks the catalogue against the gate and the suites; a gate missing one is caught.

**Left:** running a tenant's own space suites through `sb tenant verify` in a deployed cell (SB-903).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
