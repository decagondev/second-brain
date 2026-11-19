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
| Status | Open |

## Why

A template or agent change mustn't reach a tenant unless its department's suite passes.

## Acceptance criteria

- [ ] Every template ships a citation suite and a space-leak suite
- [ ] The release gate (SB-502) blocks on any regression in either
- [ ] A tenant's own space suites run in `sb tenant verify`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Suites
- [ ] Gate wiring
- [ ] Verify wiring
- [ ] Write or extend the law tests: LAW-06, LAW-20, LAW-22
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
