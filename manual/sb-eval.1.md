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
sb eval diff RESULTS/REL_A RESULTS/REL_B [--suite SUITE] [--gate FILE]
sb eval gate RESULTS/REL [--stable RESULTS/REL_STABLE]
```

## DESCRIPTION

**run** executes suites in [sb-eval(5)](sb-eval.5.md) format and prints a pass/fail table plus scores. `SUITE` is a file or a name from `evals/suites/` (`permission`, `citation`, `injection`, `refusal`, `regression`, `tenant/<name>`). With `--model mock`, the run is deterministic and offline, which is how unit CI uses it.

Each case runs three times; the **worst** result counts.

Models: `--model mock` answers from canned responses in the suite's fixture world, which is enough to test the runner and the checks. `--model extractive` (and `misattributing`, for the self-test) runs the real ask path in a local cell over the suite's `world.yaml` (sb-eval(5)). The cell is the `tools/evalcell` program, built on first use. `--model openaicompat=BASE#MODEL` runs the same local cell with answers from model `MODEL` at an OpenAI-compatible endpoint with no key, such as a model on a self-hosted cell's own machine; `make selfhost-model` gates a model that way before the cell uses it (sb-selfhost(7); SB-1710). Suites print their citation score. `--tenant-dir DIR` adds every suite in `DIR/evals/` as `tenant/<name>`. `--record FILE` writes `{"passed", "at", "suites", "model"}` after the run, which `sb tenant apply` reads to enable a workflow (sb-tenant(1)). It records a failed run too, so a failure is never mistaken for "not run yet". `--results DIR --release REL` stores each suite's result as `DIR/REL/<suite>.json`: every case's pass or fail and failed checks, the failure count, the pass rate, the mean support (citation), the model and the time. With a local model, `--release` only labels the results. Running against a deployed staging cell (`--tenant`) needs a staging cell and exits `ENOTIMPL` until one exists. Runs are deterministic: the local cell and its stand-in models have no randomness, and a suite's cases run in file order (SB-501).

**gate** applies `evals/gate.yaml` to one release's stored results (SB-502). For each suite in the gate file, it needs that suite's result, and checks `failures_max` against the failure count and `score_min` against the score: the mean support for `citation`, the pass rate for everything else. With `--stable`, it also checks `drop_max`: how far the score may fall below the current stable's. A missing result is a failure, never a pass. It prints one line per suite and exits 1 (`EEVAL`) if any suite fails. `sb release promote` refuses a release whose gate failed.

**diff** compares two releases' stored results (`--results`), suite by suite, and prints each suite's score and failures for A and B (SB-1507). B **regresses** on a suite when it misses the gate rule A met, when its score drops by more than `drop_max`, when a case A passed fails, or when B has no result for a suite A has. A suite only B has is reported, not a regression. It exits 1 (`EEVAL`) on any regression and 7 if either directory has no results.

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
