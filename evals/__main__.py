"""python3 -m evals run SUITE... [--model mock] (sb-eval(1))."""
import sys

from .runner import HERE, load, run_suite
from .targets import MockTarget


def main(argv):
    if len(argv) < 2 or argv[0] != "run":
        print("usage: python3 -m evals run SUITE... --model mock", file=sys.stderr)
        return 2
    args, suites, model = argv[1:], [], None
    i = 0
    while i < len(args):
        a = args[i]
        if a == "--model":
            model, i = args[i + 1], i + 2
        elif a in ("--tenant", "--release"):
            print(f"sb eval run: {a} not built yet (SB-501) [ENOTIMPL]", file=sys.stderr)
            return 2
        else:
            suites.append(a)
            i += 1
    if model != "mock":
        print("sb eval run: only --model mock is built in W0; real targets arrive with SB-501 [ENOTIMPL]", file=sys.stderr)
        return 2
    print(f"{'suite':<12}{'case':<14}result")
    failed = 0
    for s in suites:
        suite = load(s)
        for cid, ok, why in run_suite(suite, MockTarget(HERE / suite["world"])):
            failed += not ok
            detail = "" if ok else "  " + "; ".join(f"{n}: {d}" for n, _, d in why)
            print(f"{suite['suite']:<12}{cid:<14}{'pass' if ok else 'FAIL'}{detail}")
    if failed:
        print(f"evals: {failed} case(s) failed [EEVAL]", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
