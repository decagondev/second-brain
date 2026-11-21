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


def diff(gate, a, b, only=None):
    """Lines of (suite, regressed, detail) comparing release B's results with A's, for every
    suite either release has (or just `only`). A regression is a suite that met its gate rule
    in A and misses it in B, a score that drops more than drop_max, or a case that passed in A
    and fails in B. A suite B lacks is a regression; a suite A lacks is only reported."""
    lines = []
    for suite in sorted(set(a) | set(b)):
        if only and suite != only:
            continue
        ra, rb = a.get(suite), b.get(suite)
        if rb is None:
            lines.append((suite, True, "no results for B"))
            continue
        if ra is None:
            lines.append((suite, False, "new in B: " + ("pass" if check({suite: gate.get(suite, {})}, b)[0][1] else "FAIL")))
            continue
        rule = gate.get(suite, {})
        problems = []
        ok_a = check({suite: rule}, a)[0][1]
        ok_b, why_b = check({suite: rule}, b)[0][1:]
        if ok_a and not ok_b:
            problems.append("misses the gate now: " + why_b)
        sa, sb = score(suite, ra), score(suite, rb)
        if sa is not None and sb is not None and "drop_max" in rule and sa - sb > rule["drop_max"]:
            problems.append(f"score dropped {sa - sb:.3f} (max {rule['drop_max']})")
        passed_a = {c["id"] for c in ra.get("cases", []) if c["pass"]}
        newly = sorted(c["id"] for c in rb.get("cases", []) if not c["pass"] and c["id"] in passed_a)
        if newly:
            problems.append("newly failing: " + ", ".join(newly))
        change = "" if sa is None or sb is None else f"score {sa:.3f} -> {sb:.3f}; "
        detail = change + f"failures {ra['failures']} -> {rb['failures']}"
        lines.append((suite, bool(problems), detail + ("; " + "; ".join(problems) if problems else "")))
    return lines


def diff_main(argv):
    import sys
    if len(argv) < 2 or argv[0].startswith("-") or argv[1].startswith("-"):
        print("usage: python3 -m evals diff RESULTS/REL_A RESULTS/REL_B [--suite SUITE] [--gate FILE]", file=sys.stderr)
        return 2
    a_dir, b_dir, only, gate_file = argv[0], argv[1], None, HERE / "gate.yaml"
    i = 2
    while i < len(argv):
        if argv[i] == "--suite" and i + 1 < len(argv):
            only, i = argv[i + 1], i + 2
        elif argv[i] == "--gate" and i + 1 < len(argv):
            gate_file, i = Path(argv[i + 1]), i + 2
        else:
            print(f"evals diff: unknown option {argv[i]} [EUSAGE]", file=sys.stderr)
            return 2
    for d in (a_dir, b_dir):
        if not Path(d).is_dir():
            print(f"evals diff: {d} has no stored results [EUNAVAIL]", file=sys.stderr)
            return 7
    gate = yaml.safe_load(Path(gate_file).read_text())
    lines = diff(gate, load_results(a_dir), load_results(b_dir), only)
    for suite, bad, detail in lines:
        print(f"{suite:<26}{'REGRESSED' if bad else 'ok':<11}{detail}")
    if any(bad for _, bad, _ in lines):
        print("diff: B regresses on A [EEVAL]", file=sys.stderr)
        return 1
    return 0
