---
title: Evals
summary: The eval harness, suites, fixtures and release-gate thresholds.
---

# Evals

The eval harness from [sb-evals(7)](../manual/sb-evals.7.md) (SB-015). File formats are in [sb-eval(5)](../manual/sb-eval.5.md).

```sh
sb eval run evals/suites/smoke.yaml --model mock   # or: python3 -m evals run ... --model mock
```

| Path | What it is |
|---|---|
| [checks.py](checks.py) | The seven expectation checks. An unknown expectation fails |
| [runner.py](runner.py) | Loads suites; each case runs three times and the worst run counts |
| [targets.py](targets.py) | The mock target (canned responses) and the local cell (`tools/evalcell`). Deployed cells arrive with SB-501 |
| [suites/smoke.yaml](suites/smoke.yaml) | W0 smoke suite |
| [suites/citation.yaml](suites/citation.yaml) | Citation-faithfulness suite, run on the local cell (SB-309) |
| [faithfulness.py](faithfulness.py) | The lexical support scorer (sb-eval(5), Faithfulness) |
| [fixtures/citation/world.yaml](fixtures/citation/world.yaml) | The citation suite's world |
| [fixtures/smoke/responses.yaml](fixtures/smoke/responses.yaml) | Mock responses for the smoke suite |
| [gate.yaml](gate.yaml) | Release-gate thresholds (used from SB-502; two reviewers to change) |
| [tests/test_checks.py](tests/test_checks.py) | Unit tests for the checks and the runner |
