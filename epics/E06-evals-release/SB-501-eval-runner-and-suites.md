---
title: "SB-501: Eval runner and suites"
summary: "Permission, citation, injection and refusal suites decide releases."
---

# SB-501: Eval runner and suites

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.1 Eval gate |
| Stories | US-06.1 |
| Manual | [`sb-eval(1)`](../../manual/sb-eval.1.md), [`sb-eval(5)`](../../manual/sb-eval.5.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-09](../../manual/sb-laws.7.md#law-09), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-015](../E06-evals-release/SB-015-eval-harness-scaffold.md), [SB-309](../E06-evals-release/SB-309-citation-faithfulness-eval-set-per-tenant.md) |
| Status | In review |

## Why

Permission, citation, injection and refusal suites decide releases.

## Acceptance criteria

- [x] Suites: permission, citation, injection, refusal, regression
- [x] Deterministic seeds; results stored per release id

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Suite files
- [ ] Runner against staging cell (needs a staging cell; W7)
- [x] Result store
- [x] Write or extend the law tests: LAW-01, LAW-09, LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.1b, with SB-502): five shared suites run on the local cell with the deterministic stand-in model, each with its own world. `permission` covers denied, stale-prefilter (`viewers_now`, changed after ingest without a sync) and timed-out documents. `injection` uses hidden spans and HTML comments carrying a canary and an instruction to call a tool. `refusal` covers an asker who can open nothing and a source that errors. `regression` starts with BUG-0007. The local cell gained per-case `inject: { acl: … }` faults and `viewers_now`. `--results DIR --release REL` stores each suite as `DIR/REL/<suite>.json` (cases, failures, pass rate, mean support, model, time). `make evals` runs all five and gates them. Building the injection world showed that a canary between zero-width characters is visible text, not hidden, so the world now hides it in an HTML comment, as an attacker would. LAW-01, LAW-03, LAW-09 and LAW-14 are measured by these suites on top of the law tests.

**Left:** the same suites against a staging cell with the release's model (W7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
