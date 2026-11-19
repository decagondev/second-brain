#!/usr/bin/env python3
"""SB-604: every step of the onboarding runbook says what to expect, and every phase ends
in a gate with a checklist (sb-onboarding(7)). A self-test runs first (see BUG-0005)."""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PHASE = re.compile(r"^### (Day 0|Days \d+.\d+)", re.M)
STEP = re.compile(r"^\d+\. (.+)$", re.M)


def problems_in(text):
    out = []
    parts = PHASE.split(text)
    phases = [(parts[i], parts[i + 1]) for i in range(1, len(parts) - 1, 2)]
    if not phases:
        return ["no phases found"]
    for name, body in phases:
        main = body.split("#### ")[0]  # sub-sections (the pilot review) have their own rules
        for step in STEP.findall(main):
            if "expect" not in step.lower():
                out.append(f"{name}: step without an expected result: {step[:60]}")
        if "**Gate" not in main:
            out.append(f"{name}: no gate")
        elif name != "Day 0" and "- [ ]" not in main:
            out.append(f"{name}: no checklist")
    return out


def self_test():
    bad = "### Day 0\n1. Do a thing.\n**Gate 0:** x\n"
    good = "### Days 1-14\n1. Do a thing. -> expect y.\n**Gate 1:** x\n\n- [ ] y\n"
    assert problems_in(bad), "runbook.py self-test: rule does not match"
    assert problems_in(good) == [], "runbook.py self-test: rule matches good text"


def main():
    self_test()
    text = (ROOT / "manual" / "sb-onboarding.7.md").read_text(encoding="utf-8")
    ps = problems_in(text)
    for p in ps:
        print("runbook:", p)
    print(f"runbook: {len(ps)} problems")
    return 1 if ps else 0


if __name__ == "__main__":
    sys.exit(main())
