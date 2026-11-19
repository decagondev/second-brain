"""python3 -m evals run SUITE... [--model mock|extractive|misattributing] [--tenant-dir DIR] (sb-eval(1))."""
import sys

import yaml

from .runner import HERE, RUNS, load, run_suite, tenant_suites
from .targets import LocalCell, MockTarget

MODELS = ("mock", "extractive", "misattributing")


def main(argv):
    if argv and argv[0] == "gate":
        from .gate import main as gate_main
        return gate_main(argv[1:])
    if len(argv) < 2 or argv[0] != "run":
        print("usage: python3 -m evals run SUITE... --model mock|extractive [--tenant-dir DIR]", file=sys.stderr)
        return 2
    args, suites, model, record, results_dir, release = argv[1:], [], None, None, None, None
    i = 0
    try:
        while i < len(args):
            a = args[i]
            if a == "--model":
                model, i = args[i + 1], i + 2
            elif a == "--record":
                record, i = args[i + 1], i + 2
            elif a == "--tenant-dir":
                shared = yaml.safe_load((HERE / "gate.yaml").read_text())["citation"]["score_min"]
                suites.extend(tenant_suites(args[i + 1], shared))
                i += 2
            elif a == "--results":
                results_dir, i = args[i + 1], i + 2
            elif a == "--release":
                release, i = args[i + 1], i + 2
            elif a == "--tenant":
                print("sb eval run: --tenant needs a staging cell, which doesn't exist yet [ENOTIMPL]", file=sys.stderr)
                return 2
            else:
                suites.append(load(a))
                i += 1
    except (ValueError, IndexError, FileNotFoundError) as e:
        print(f"sb eval run: {e} [EUSAGE]", file=sys.stderr)
        return 2
    if (results_dir is None) != (release is None):
        print("sb eval run: --results and --release go together [EUSAGE]", file=sys.stderr)
        return 2
    if model not in MODELS:
        print(f"sb eval run: --model must be one of {', '.join(MODELS)}; deployed cells arrive with SB-501 [ENOTIMPL]", file=sys.stderr)
        return 2
    print(f"{'suite':<26}{'case':<14}result")
    failed = 0
    for suite in suites:
        world = suite.get("_dir", HERE) / suite["world"]
        target = MockTarget(world) if model == "mock" else LocalCell(world, model, RUNS)
        rows, score = run_suite(suite, target)
        if results_dir:
            from .gate import result, store
            store(results_dir, release or "rel_local", result(suite["suite"], release or "rel_local", model, rows, score))
        for cid, ok, why in rows:
            failed += not ok
            detail = "" if ok else "  " + "; ".join(f"{n}: {d}" for n, _, d in why)
            print(f"{suite['suite']:<26}{cid:<14}{'pass' if ok else 'FAIL'}{detail}")
        gate = suite.get("gate", {}).get("score_min")
        if score is not None:
            line = f"{suite['suite']:<26}{'score':<14}{score:.3f}"
            if gate is not None:
                line += f" (min {gate})"
                if score < gate:
                    failed += 1
                    line += " FAIL"
            print(line)
    if record:
        write_record(record, failed == 0, [x["suite"] for x in suites], model)
    if failed:
        print(f"evals: {failed} case(s) or score(s) failed [EEVAL]", file=sys.stderr)
        return 1
    return 0


def write_record(path, passed, suites, model):
    """The eval record sb tenant apply reads to enable a workflow (sb-tenant(1))."""
    import datetime
    import json
    import os
    from pathlib import Path
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    at = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    tmp = p.with_suffix(p.suffix + ".tmp")
    tmp.write_text(json.dumps({"passed": passed, "at": at, "suites": sorted(set(suites)), "model": model}) + "\n")
    os.replace(tmp, p)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
