#!/usr/bin/env python3
"""SB-807: eval suites that are red on purpose fail for the right reason, and only for it.

For each entry in evals/expected_red.yaml, run the suite on the local cell (extractive model)
and require that exactly the listed cases fail. A listed case that passes means the ticket has
landed (remove the entry); an unlisted case that fails means the suite is red for the wrong
reason. A self-test runs first (see BUG-0005).
"""
import re
import subprocess
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
ROW = re.compile(r"^(\S+)\s+(\S+?)\s*(pass|FAIL)\b", re.M)


def outcome(text):
    """case id -> passed, from the runner's table."""
    return {m.group(2): m.group(3) == "pass" for m in ROW.finditer(text) if m.group(2) != "score"}


def problems(name, entry, results):
    out = []
    want = set(entry["failing"])
    if not results:
        return [f"{name}: the suite produced no results"]
    for cid, passed in sorted(results.items()):
        if cid in want and passed:
            out.append(f"{name}: {cid} passes; has {entry['ticket']} landed? Remove the entry and add the suite to the gate")
        if cid not in want and not passed:
            out.append(f"{name}: {cid} fails, but it isn't one of the expected failures: red for the wrong reason")
    for cid in want - set(results):
        out.append(f"{name}: {cid} is listed but isn't in the suite")
    return out


def self_test():
    table = "s  a-1          FAIL  x\ns  b-1          pass\ns  score        1.000\n"
    r = outcome(table)
    assert r == {"a-1": False, "b-1": True}, "evals_red.py self-test: table parsing"
    assert problems("s", {"ticket": "T", "failing": ["a-1"]}, r) == [], "self-test: right reason flagged"
    assert problems("s", {"ticket": "T", "failing": ["b-1"]}, r), "self-test: wrong reason not flagged"


def main():
    self_test()
    reg = yaml.safe_load((ROOT / "evals" / "expected_red.yaml").read_text()) or {}
    all_problems = []
    for name, entry in reg.items():
        p = subprocess.run([sys.executable, "-m", "evals", "run", entry["suite"], "--model", "extractive"],
                           cwd=ROOT, capture_output=True, text=True)
        all_problems += problems(name, entry, outcome(p.stdout))
    for p in all_problems:
        print("evals-red:", p)
    print(f"evals-red: {len(reg)} suite(s) red for the right reason" if not all_problems else f"evals-red: {len(all_problems)} problems")
    return 1 if all_problems else 0


if __name__ == "__main__":
    sys.exit(main())
