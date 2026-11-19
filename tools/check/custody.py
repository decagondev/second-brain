#!/usr/bin/env python3
"""SB-114: code learns where a cell lives only from its custody record (ADR-0015).

Fails on any of these in non-test, non-vendor Go code outside internal/core/cell/custody.go:
  - a cloud account address (`tenant-account://...`) or a 12-digit account id literal;
  - the custody literals "factory-account", or Custody/Placement/KeyCustodian fields set
    from a string literal instead of the cell constants.
A self-test runs first, so the rule can't silently stop matching (see BUG-0005).
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ALLOWED = "internal/core/cell/custody.go"
RULES = [
    (re.compile(r"tenant-account://"), "a cloud account address"),
    (re.compile(r'"\d{12}"'), "a 12-digit account id"),
    (re.compile(r'"factory-account"'), 'the placement literal "factory-account"'),
    (re.compile(r'\b(Custody|Placement|KeyCustodian)\s*:\s*"'), "a custody field set from a literal"),
]


def problems_in(rel, text):
    out = []
    for n, line in enumerate(text.splitlines(), 1):
        if line.lstrip().startswith("//"):
            continue
        for rx, what in RULES:
            if rx.search(line):
                out.append(f"{rel}:{n}: {what}; read it from the custody record (cell constants, ADR-0015)")
    return out


def self_test():
    bad = 'a := "factory-account"\nb := "tenant-account://x"\nc := "123456789012"\nr := CellRecord{Custody: "factory"}\n'
    good = "r := CellRecord{Custody: cell.CustodyFactory}\n// tenant-account://example in a comment\n"
    assert len(problems_in("x.go", bad)) == 4, "custody.py self-test: rules do not match"
    assert problems_in("x.go", good) == [], "custody.py self-test: rules match good code"


if __name__ == "__main__":
    self_test()
    found = []
    for f in sorted(ROOT.rglob("*.go")):
        rel = f.relative_to(ROOT).as_posix()
        if rel.startswith(("vendor/", "site/")) or rel.endswith("_test.go") or rel == ALLOWED:
            continue
        found += problems_in(rel, f.read_text(encoding="utf-8"))
    for p in found:
        print("custody: " + p)
    print(f"custody: {len(found)} problems")
    sys.exit(1 if found else 0)
