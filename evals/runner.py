"""Run suites; each case runs three times and the worst run counts (sb-eval(1))."""
from pathlib import Path

import yaml

from .checks import evaluate

RUNS = 3
HERE = Path(__file__).resolve().parent


def load(path):
    p = Path(path)
    if not p.exists():
        p = HERE / "suites" / f"{path}.yaml"
    suite = yaml.safe_load(p.read_text())
    for key in ("suite", "world", "cases"):
        if key not in suite:
            raise ValueError(f"{p}: missing {key!r} (sb-eval(5))")
    ids = [c["id"] for c in suite["cases"]]
    if len(ids) != len(set(ids)):
        raise ValueError(f"{p}: duplicate case ids")
    return suite


def run_suite(suite, target):
    rows = []
    for case in suite["cases"]:
        worst = None
        for attempt in range(RUNS):
            results = evaluate(target.run(case, attempt), case.get("expect", {}))
            failed = [r for r in results if not r[1]]
            if worst is None or len(failed) > len(worst):
                worst = failed
        rows.append((case["id"], not worst, worst))
    return rows
