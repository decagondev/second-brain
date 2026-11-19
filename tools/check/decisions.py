#!/usr/bin/env python3
"""SB-109: callers must treat policy.Unknown as refusal (LAW-03, sb-permissions(7)).

Fails on, in any non-vendor Go file:
  1. a `!= policy.Deny` (or `!= Deny` inside package policy) test, which lets Unknown through;
  2. a `default:` case in a switch on a Decision, which hides a missing Unknown case.
The `exhaustive` linter separately requires every Decision switch to name Unknown.
A line may opt out with `//nolint:decisions` (used only by the law self-test's mutation).
Runs its own self-test first, so the rule can't silently stop matching (see BUG-0005).
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
NOT_DENY = re.compile(r"!=\s*(policy\.)?Deny\b")
SWITCH = re.compile(r"\bswitch\b[^{\n]*\bDecision\b[^{\n]*\{")


def problems_in(rel, text):
    out = []
    for n, line in enumerate(text.splitlines(), 1):
        if "//nolint:decisions" in line:
            continue
        if NOT_DENY.search(line) and ("policy." in line or "/core/policy/" in rel):
            out.append(f"{rel}:{n}: `!= Deny` lets Unknown through; test `== policy.Allow` or use policy.Require")
    for m in SWITCH.finditer(text):
        depth, i = 1, m.end()
        while i < len(text) and depth:
            c = text[i]
            if c == "{":
                depth += 1
            elif c == "}":
                depth -= 1
            elif depth == 1 and text.startswith("default:", i):
                n = text.count("\n", 0, i) + 1
                out.append(f"{rel}:{n}: `default:` in a switch on a Decision; name Allow, Deny and Unknown")
            i += 1
    return out


def self_test():
    bad = "if d != policy.Deny {\n}\nswitch r.Decision {\ncase policy.Allow:\ndefault:\n}\n"
    good = "if d == policy.Allow {\n}\nswitch r.Decision {\ncase policy.Allow:\ncase policy.Deny, policy.Unknown:\n}\n"
    assert len(problems_in("x.go", bad)) == 2, "decisions.py self-test: rule does not match"
    assert problems_in("x.go", good) == [], "decisions.py self-test: rule matches good code"


if __name__ == "__main__":
    self_test()
    found = []
    for f in sorted(ROOT.rglob("*.go")):
        rel = f.relative_to(ROOT).as_posix()
        if rel.startswith(("vendor/", "site/")):
            continue
        found += problems_in(rel, f.read_text(encoding="utf-8"))
    for p in found:
        print("decisions: " + p)
    print(f"decisions: {len(found)} problems")
    sys.exit(1 if found else 0)
