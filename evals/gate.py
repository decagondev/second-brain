"""Stored results per release and the release gate (sb-eval(1), gate; SB-501, SB-502).

A missing result is a failure, never a pass, and there is no override."""
import datetime
import json
from pathlib import Path

import yaml

HERE = Path(__file__).resolve().parent

# Which number a suite's score_min and drop_max apply to.
SCORE_FIELD = {"citation": "support"}


def result(suite, release, model, rows, support):
    cases = [{"id": cid, "pass": ok, "failed": [n for n, _, _ in why]} for cid, ok, why in rows]
    failures = sum(not c["pass"] for c in cases)
    return {
        "suite": suite, "release": release, "model": model,
        "at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "cases": cases, "failures": failures,
        "pass_rate": (len(cases) - failures) / len(cases) if cases else 0.0,
        "support": support,
    }


def store(results_dir, release, res):
    d = Path(results_dir) / release
    d.mkdir(parents=True, exist_ok=True)
    name = res["suite"].replace("/", "_") + ".json"
    (d / name).write_text(json.dumps(res, indent=1, sort_keys=True) + "\n")


def load_results(d):
    out = {}
    for f in sorted(Path(d).glob("*.json")):
        r = json.loads(f.read_text())
        out[r["suite"]] = r
    return out


def score(suite, r):
    v = r.get(SCORE_FIELD.get(suite, "pass_rate"))
    return float(v) if v is not None else None


def check(gate, results, stable=None):
    """Lines of (suite, ok, detail) for every suite in the gate file."""
    lines = []
    for suite, rule in gate.items():
        r = results.get(suite)
        if r is None:
            lines.append((suite, False, "no results"))
            continue
        problems = []
        if "failures_max" in rule and r["failures"] > rule["failures_max"]:
            problems.append(f"{r['failures']} failures (max {rule['failures_max']})")
        s = score(suite, r)
        if "score_min" in rule and (s is None or s < rule["score_min"]):
            problems.append(f"score {'none' if s is None else f'{s:.3f}'} (min {rule['score_min']})")
        if "drop_max" in rule and stable is not None and suite in stable:
            prev = score(suite, stable[suite])
            if prev is not None and s is not None and prev - s > rule["drop_max"]:
                problems.append(f"score dropped {prev - s:.3f} from stable {prev:.3f} (max {rule['drop_max']})")
        lines.append((suite, not problems, "; ".join(problems) or (f"score {s:.3f}" if s is not None else "ok")))
    return lines


def main(argv):
    if not argv or argv[0].startswith("-"):
        print("usage: python3 -m evals gate RESULTS/REL [--stable RESULTS/REL] [--gate FILE]", file=__import__("sys").stderr)
        return 2
    results_dir, stable_dir, gate_file = argv[0], None, HERE / "gate.yaml"
    i = 1
    while i < len(argv):
        if argv[i] == "--stable":
            stable_dir, i = argv[i + 1], i + 2
        elif argv[i] == "--gate":
            gate_file, i = Path(argv[i + 1]), i + 2
        else:
            print(f"evals gate: unknown option {argv[i]} [EUSAGE]", file=__import__("sys").stderr)
            return 2
    gate = yaml.safe_load(Path(gate_file).read_text())
    lines = check(gate, load_results(results_dir), load_results(stable_dir) if stable_dir else None)
    for suite, ok, detail in lines:
        print(f"{suite:<14}{'pass' if ok else 'FAIL'}  {detail}")
    if not all(ok for _, ok, _ in lines):
        print("gate: the release doesn't pass [EEVAL]", file=__import__("sys").stderr)
        return 1
    return 0
