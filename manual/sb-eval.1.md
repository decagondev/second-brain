---
title: "sb-eval(1)"
man: "1"
summary: Run eval suites and compare results between releases.
---

# sb-eval(1)

## NAME

sb-eval — run evaluation suites against a cell or a mock model

## SYNOPSIS

```text
sb eval run  SUITE... [--tenant T] [--release REL] [--model mock]
sb eval diff REL_A REL_B [--suite SUITE]
```

## DESCRIPTION

**run** executes suites in [sb-eval(5)](sb-eval.5.md) format and prints a pass/fail table plus scores. `SUITE` is a file or a name from `evals/suites/` (`permission`, `citation`, `injection`, `refusal`, `regression`, `tenant/<name>`). With `--model mock`, the run is deterministic and offline, which is how unit CI uses it.

Each case runs three times; the **worst** result counts.

Models: `--model mock` answers from canned responses in the suite's fixture world, which is enough to test the runner and the checks. `--model extractive` (and `misattributing`, for the self-test) runs the real ask path in a local cell over the suite's `world.yaml` (sb-eval(5)). The cell is the `tools/evalcell` program, built on first use. Suites print their citation score. `--tenant-dir DIR` adds every suite in `DIR/evals/` as `tenant/<name>`. `--record FILE` writes `{"passed", "at", "suites", "model"}` after the run, which `sb tenant apply` reads to enable a workflow (sb-tenant(1)). It records a failed run too, so a failure is never mistaken for "not run yet". Running against a deployed cell (`--tenant`, `--release`) arrives with SB-501; until then those options exit with `ENOTIMPL`.

**diff** compares stored results for two releases and highlights regressions against the thresholds in `evals/gate.yaml`.

## EXIT STATUS

0 all suites meet thresholds · 1 `EEVAL`: a threshold was missed · 7 target unavailable.

## LAWS

[LAW-09](sb-laws.7.md#law-09), [LAW-14](sb-laws.7.md#law-14).

## EXAMPLES

```text
$ sb eval run injection citation --tenant staging-1
suite       cases  pass  score   threshold  result
injection   120    120   1.000   1.000      pass
citation    200    193   0.941   0.920      pass
```

## SEE ALSO

[sb-evals(7)](sb-evals.7.md), [sb-eval(5)](sb-eval.5.md)
