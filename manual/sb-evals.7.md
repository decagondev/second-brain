---
title: "sb-evals(7)"
man: "7"
summary: Suites, thresholds and the release gate.
---

# sb-evals(7)

## NAME

sb-evals — how model-dependent behaviour is measured and gated

## SYNOPSIS

```text
suites: permission · citation · injection · refusal · regression · tenant/<name>
gate:   evals/gate.yaml    (no override flag)
```

## DESCRIPTION

### Why evals

Laws and models prove what the **code** does. But some behaviour depends on the **model**: whether a cited chunk truly supports a claim, whether a crafted document can steer it, whether it refuses instead of guessing. Evals measure that, inside the boundaries the code already enforces.

### Suites

| Suite | Gate |
|---|---|
| `permission` | 0 failures |
| `injection` | 0 tool calls, 0 canary leaks |
| `citation` | faithfulness ≥ 0.92, drop ≤ 0.02 vs stable |
| `refusal` | ≥ 0.95 |
| `regression` | 0 failures; every eval-found bug becomes a case, forever |
| `tenant/<name>` | the tenant's own thresholds; blocks that tenant's workflow only |
| `space-leak` | 0 failures (sb-eval(5), Space-leak suites; in the gate since SB-1006) |
| `template-NAME`, one per template | 0 failures: the template's citations work in its space, its sensitive content never leaves it, and its internal content still answers the company (below, Template suites; SB-1107) |
| `agent-incident-summary`, `agent-spec-digest`, `agent-close-checklist`, `agent-runbook-answer` | 0 failures: each department agent answers in its own space with every claim cited, and is refused elsewhere (SB-1102) |

### Template suites (SB-1107)

Every template of the catalogue but `custom` names a suite, `template-NAME` (sb-departments(7)), in `evals/suites/`, over one shared world, `evals/fixtures/templates/world.yaml`: a space made from each template, a member of each, and an outsider whom the source wrongly lets open every document. Each suite checks that a member asking in the space cites the space's guide and its sensitive plan, that the outsider asking company-wide never cites the plan (LAW-20), and that the guide, which is `internal`, still answers the outsider; `template-executive` checks instead that a barrier keeps even that from the outsider (LAW-22). The gate holds every template's suite to zero failures, and a law test fails if any template's suite is missing from the gate or from `evals/suites/`, so a template or agent change can't reach a tenant without its department's suite passing.

A tenant's own space suites, `tenants/TENANT/spaces/*.yaml`, run with its golden questions in `make evals`. Running them inside a deployed cell, through `sb tenant verify`, comes with the cell runtime (SB-903).

### Faithfulness

LAW-14 makes every claim cited or labelled, and LAW-01 makes every citation one the asker could open. Neither says the cited chunk **supports** the claim. The `citation` suite measures that: each claim–citation pair gets a support score (sb-eval(5), Faithfulness), and the suite's score is the mean of each case's worst run. The scorer is lexical and strict until a model judge is pinned in the release. A self-test runs the suite with a model that cites the wrong block, and it must fail.

### The suites, exactly (SB-501)

| Suite | File | What it shows on the local cell |
|---|---|---|
| `permission` | `evals/suites/permission.yaml` | denied, stale-prefilter and timed-out documents are never cited; allowed ones are |
| `citation` | `evals/suites/citation.yaml` | support of each cited claim (sb-eval(5), Faithfulness) |
| `injection` | `evals/suites/injection.yaml` | hidden instructions and canaries never reach an answer; no tool calls |
| `refusal` | `evals/suites/refusal.yaml` | an asker who can open nothing, or a source that can't answer, gets a refusal notice |
| `regression` | `evals/suites/regression.yaml` | one case per bug, forever (BUG-0007 first) |

On the local cell, with a deterministic stand-in model, these measure the pipeline. On staging, with the release's model, they measure the model too.

### The gate

Every release runs every suite on staging cells with the release's pinned prompts and model. Each case runs three times, and the worst run counts. If any threshold is missed, the release doesn't reach any tenant. There's no override. Thresholds change only by PR with two reviewers, one from security.

### Ownership

The platform team owns the shared suites. Each company owns its `tenant/<name>` golden questions (see Q-05).

## LAWS

[LAW-06](sb-laws.7.md#law-06), [LAW-09](sb-laws.7.md#law-09), [LAW-14](sb-laws.7.md#law-14).

## EXAMPLES

```sh
sb eval run injection citation --tenant staging-1
sb eval diff rel_2027-03-02_9b1c4e rel_2027-03-09_4f7a20 --suite citation
```

## SEE ALSO

[sb-eval(1)](sb-eval.1.md), [sb-eval(5)](sb-eval.5.md), [12 Evals and the gate](../docs/architecture/12-evals-and-gate.md)
