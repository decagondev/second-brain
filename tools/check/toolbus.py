#!/usr/bin/env python3
"""SB-405: no tool can be invoked except through the bus (sb-agents(7), The tool bus).

Fails on any call of a connector's Act (`.Act(`) in non-test, non-vendor Go code outside
internal/core/toolbus. The connector contract kit (internal/core/connectorkit) and the law
harness (laws/) exercise Act on purpose and are exempt, like tests. A self-test runs first,
so the rule can't silently stop matching (see BUG-0005).
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EXEMPT = ("vendor/", "internal/core/toolbus/", "internal/core/connectorkit/", "laws/")
CALL = re.compile(r"\.Act\(")


def problems_in(rel, text):
    out = []
    for n, line in enumerate(text.splitlines(), 1):
        code = line.split("//", 1)[0]
        if CALL.search(code) and not re.search(r"\bfunc\s*\([^)]*\)\s*Act\(", code):
            out.append(f"{rel}:{n}: calls Act outside the tool bus (SB-405)")
    return out


def self_test():
    bad = "x := conn.Act(ctx, a, key)\n"
    good = "func (c *Conn) Act(ctx context.Context, a ports.Action, k ports.IdempotencyKey) (ports.Receipt, error) {\n// c.Act( in a comment\n"
    assert len(problems_in("x.go", bad)) == 1, "toolbus.py self-test: rule does not match"
    assert problems_in("x.go", good) == [], "toolbus.py self-test: rule matches good code"


def main():
    self_test()
    problems = []
    for f in sorted(ROOT.rglob("*.go")):
        rel = f.relative_to(ROOT).as_posix()
        if rel.startswith(EXEMPT) or rel.endswith("_test.go") or "/node_modules/" in rel:
            continue
        problems += problems_in(rel, f.read_text(encoding="utf-8"))
    for p in problems:
        print("toolbus:", p)
    print(f"toolbus: {len(problems)} problems")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
