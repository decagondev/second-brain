#!/usr/bin/env python3
"""SB-010: the law ratchet.

Runs `go test -json ./laws/...` and checks every test against laws/expected_red.txt:
- a law listed there must FAIL, and only with "not implemented" (failing for the right reason).
  A line is `LAW-nn TICKET` (every test of that law) or `LAW-nn TICKET TestName` (that test only),
  so one law can have a green test at one level and a red one at another (e.g. LAW-02);
- a listed law that PASSES is an error too: remove it from the list and update the ledger;
- every other test (other laws, self-tests) must PASS.
"""
import json, re, subprocess, sys
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
entries = []  # (law, ticket, test or None)
for line in (ROOT / "laws/expected_red.txt").read_text().splitlines():
    if line.strip() and not line.startswith("#"):
        parts = line.split()
        entries.append((parts[0], parts[1], parts[2] if len(parts) > 2 else None))

def expected_red(t, law):
    for e in entries:
        if e[0] == law and e[2] == t:
            return e
    for e in entries:
        if e[0] == law and e[2] is None:
            return e
    return None

p = subprocess.run(["go", "test", "-json", "-count=1", "./laws/..."], cwd=ROOT, capture_output=True, text=True)
result, output = {}, defaultdict(str)
for line in p.stdout.splitlines():
    try:
        ev = json.loads(line)
    except json.JSONDecodeError:
        continue
    t = ev.get("Test")
    if not t or "/" in t:
        continue
    if ev["Action"] in ("pass", "fail"):
        result[t] = ev["Action"]
    elif ev["Action"] == "output":
        output[t] += ev.get("Output", "")
if not result:
    sys.exit("laws.py: no tests ran\n" + p.stderr)

problems, rows = [], []
for t in sorted(result):
    m = re.match(r"TestLAW(\d\d)_", t)
    law = f"LAW-{m.group(1)}" if m else None
    e = expected_red(t, law)
    if e:
        if result[t] == "pass":
            problems.append(f"{t} ({law}) is green: remove its line from laws/expected_red.txt and update the ledger")
        elif "not implemented" not in output[t]:
            problems.append(f"{t} ({law}) is red for the wrong reason:\n{output[t].strip()}")
        rows.append((t, "red (expected, " + e[1] + ")"))
    else:
        if result[t] != "pass":
            problems.append(f"{t} failed:\n{output[t].strip()}")
        rows.append((t, result[t]))
for law, ticket, test in entries:
    if test and test not in result:
        problems.append(f"{test} ({law}) is in expected_red.txt but has no test")
    elif not test and not any(t.startswith(f"TestLAW{law[4:]}_") for t in result):
        problems.append(f"{law} is in expected_red.txt but has no test")
w = max(len(t) for t, _ in rows)
for t, s in rows:
    print(f"  {t:<{w}}  {s}")
if problems:
    print("\n".join("laws: " + x for x in problems))
    sys.exit(1)
print(f"laws: {len(rows)} tests; {len(entries)} expected red, all failing for the right reason; the rest green")
