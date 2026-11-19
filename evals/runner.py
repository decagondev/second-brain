"""Run suites; each case runs three times and the worst run counts (sb-eval(1))."""
from pathlib import Path

import yaml

from .checks import evaluate, worst_support

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
    """Rows (case id, passed, failed checks) and the suite's citation score: the mean over
    cases of each case's worst-run lowest claim support (None if nothing was cited)."""
    if hasattr(target, "prepare"):
        target.prepare(suite["cases"])
    rows, supports = [], []
    for case in suite["cases"]:
        worst, low = None, None
        for attempt in range(RUNS):
            res = target.run(case, attempt)
            results = evaluate(res, case.get("expect", {}))
            failed = [r for r in results if not r[1]]
            if worst is None or len(failed) > len(worst):
                worst = failed
            s = worst_support(res)
            if s is not None and (low is None or s < low):
                low = s
        rows.append((case["id"], not worst, worst))
        if low is not None:
            supports.append(low)
    score = sum(supports) / len(supports) if supports else None
    return rows, score


def tenant_suites(tenant_dir, shared_min):
    """Every suite in tenant_dir/evals, named tenant/<name>. A tenant gate may only be
    stricter than the shared citation threshold (sb-eval(5), Tenant golden questions)."""
    d = Path(tenant_dir)
    name = d.name
    out = []
    for f in sorted((d / "evals").glob("*.yaml")):
        if f.name == "world.yaml":
            continue
        suite = load(f)
        suite["suite"] = f"tenant/{name}"
        suite["_dir"] = f.parent
        gate = suite.get("gate", {})
        if "score_min" in gate and gate["score_min"] < shared_min:
            raise ValueError(f"{f}: gate.score_min {gate['score_min']} is looser than the shared citation threshold {shared_min}")
        out.append(suite)
    if not out:
        raise ValueError(f"{d}: no suites in {d / 'evals'}")
    return out
