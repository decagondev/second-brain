#!/usr/bin/env python3
"""SB-612: the pen-test findings register (security/findings.yaml) is well formed, every
fixed finding names its bug entry and appears in the threat model, and no critical or high
finding is open. A self-test runs first (see BUG-0005)."""
import re
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
SEVERITIES = {"critical", "high", "medium", "low", "info"}
STATUSES = {"open", "fixed", "accepted"}
FIELDS = {"id", "title", "severity", "status", "laws", "found", "bug", "source"}
ID = re.compile(r"^PT-\d{4}$")
LAW = re.compile(r"^LAW-\d{2}$")
BUG = re.compile(r"^BUG-\d{4}$")
DATE = re.compile(r"^\d{4}-\d{2}-\d{2}$")


def problems_in(findings, bugs_text, threat_text):
    out = []
    if not isinstance(findings, list):
        return ["findings must be a list"]
    seen = set()
    for i, f in enumerate(findings):
        where = f"finding {i + 1}"
        if not isinstance(f, dict):
            out.append(f"{where}: not a mapping")
            continue
        unknown = set(f) - FIELDS
        missing = {"id", "title", "severity", "status", "laws", "found"} - set(f)
        if unknown:
            out.append(f"{where}: unknown fields {sorted(unknown)}")
        if missing:
            out.append(f"{where}: missing {sorted(missing)}")
            continue
        fid = str(f["id"])
        where = fid
        if not ID.match(fid):
            out.append(f"{where}: id must be PT-nnnn")
        if fid in seen:
            out.append(f"{where}: duplicate id")
        seen.add(fid)
        sev, st = f["severity"], f["status"]
        if sev not in SEVERITIES:
            out.append(f"{where}: severity must be one of {sorted(SEVERITIES)}")
        if st not in STATUSES:
            out.append(f"{where}: status must be one of {sorted(STATUSES)}")
        if not DATE.match(str(f["found"])):
            out.append(f"{where}: found must be YYYY-MM-DD")
        laws = f["laws"] if isinstance(f["laws"], list) else []
        if not laws or not all(isinstance(x, str) and LAW.match(x) for x in laws):
            out.append(f"{where}: laws must be a non-empty list of LAW-nn")
        if sev in {"critical", "high"} and st != "fixed":
            out.append(f"{where}: a {sev} finding is {st}; it must be fixed before scale-out")
        if st == "fixed":
            bug = str(f.get("bug", ""))
            if not BUG.match(bug):
                out.append(f"{where}: a fixed finding names its bug entry (BUG-nnnn)")
            elif f"## {bug} " not in bugs_text:
                out.append(f"{where}: {bug} is not in living/BUGS_AND_MITIGATIONS.md")
            if fid not in threat_text:
                out.append(f"{where}: not mentioned in docs/07-threat-model.md")
    return out


def self_test():
    good = [{"id": "PT-0001", "title": "t", "severity": "high", "status": "fixed", "laws": ["LAW-07"], "found": "2026-11-19", "bug": "BUG-0010"}]
    bugs, threat = "## BUG-0010 x\n", "PT-0001"
    assert problems_in(good, bugs, threat) == [], "findings.py self-test: rule rejects a good register"
    bad = [dict(good[0], status="open")]
    assert problems_in(bad, bugs, threat), "findings.py self-test: an open high finding passed"
    assert problems_in([dict(good[0], bug="BUG-0099")], bugs, threat), "findings.py self-test: a missing bug entry passed"
    assert problems_in([dict(good[0])], bugs, ""), "findings.py self-test: a finding missing from the threat model passed"


def main():
    self_test()
    reg = yaml.safe_load((ROOT / "security" / "findings.yaml").read_text(encoding="utf-8")) or {}
    findings = reg.get("findings", []) if isinstance(reg, dict) else None
    bugs = (ROOT / "living" / "BUGS_AND_MITIGATIONS.md").read_text(encoding="utf-8")
    threat = (ROOT / "docs" / "07-threat-model.md").read_text(encoding="utf-8")
    ps = problems_in(findings, bugs, threat)
    for p in ps:
        print("findings:", p)
    print(f"findings: {len(findings or [])} findings, {len(ps)} problems")
    return 1 if ps else 0


if __name__ == "__main__":
    sys.exit(main())
