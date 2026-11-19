#!/usr/bin/env python3
"""SB-009: the manual can't lie.

1. Every man page has NAME, SYNOPSIS, DESCRIPTION and SEE ALSO, in that order.
2. Every code block tagged `run` in the manual is executed:
   - yaml blocks must parse; tenant manifests also go through `sb tenant check` (SB-101).
3. Every YAML file under examples/ and evals/ parses. Every manifest in examples/tenants
   passes `sb tenant check`, except invalid.yaml, whose output must equal
   invalid.expected exactly.
"""
import re, subprocess, sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
REQUIRED = ["NAME", "SYNOPSIS", "DESCRIPTION", "SEE ALSO"]
problems, pending, ran = [], set(), 0

def sb(*args, cwd=ROOT):
    exe = ROOT / "bin/sb"
    if not getattr(sb, "built", False):  # rebuild once per run: a stale binary would hide changes
        subprocess.run(["go", "build", "-o", str(exe), "./cmd/sb"], cwd=ROOT, check=True)
        sb.built = True
    return subprocess.run([str(exe), *args], capture_output=True, text=True, cwd=cwd)

for page in sorted((ROOT / "manual").glob("sb*.md")):
    text = page.read_text()
    heads = re.findall(r"^## (.+)$", text, re.M)
    idx = [heads.index(h) if h in heads else -1 for h in REQUIRED]
    if -1 in idx or idx != sorted(idx):
        problems.append(f"{page.name}: sections must include {REQUIRED} in order; found {heads}")
    for lang, body in re.findall(r"^```(\w+) run\n(.*?)^```", text, re.M | re.S):
        ran += 1
        if lang != "yaml":
            problems.append(f"{page.name}: no runner for `{lang} run` blocks")
            continue
        try:
            doc = yaml.safe_load(body)
        except yaml.YAMLError as e:
            problems.append(f"{page.name}: run block does not parse: {e}")
            continue
        if isinstance(doc, dict) and "tenant" in doc:
            tmp = ROOT / "bin" / "example-manifest.yaml"
            tmp.parent.mkdir(exist_ok=True)
            tmp.write_text(body)
            r = sb("tenant", "check", str(tmp))
            if r.returncode != 0:
                problems.append(f"{page.name}: manifest example fails sb tenant check:\n{r.stdout}{r.stderr}")

for f in sorted(list((ROOT / "examples").rglob("*.yaml")) + list((ROOT / "evals").rglob("*.yaml"))):
    try:
        yaml.safe_load(f.read_text())
    except yaml.YAMLError as e:
        problems.append(f"{f.relative_to(ROOT)}: does not parse: {e}")
    ran += 1
tenants = ROOT / "examples/tenants"
for f in sorted(tenants.glob("*.yaml")):
    r = sb("tenant", "check", f.name, cwd=tenants)
    ran += 1
    if f.name == "invalid.yaml":
        if r.returncode != 1 or r.stdout != (tenants / "invalid.expected").read_text():
            problems.append(f"examples/tenants/invalid.yaml: output differs from invalid.expected (exit {r.returncode}):\n{r.stdout}")
    elif r.returncode != 0:
        problems.append(f"examples/tenants/{f.name}: fails sb tenant check:\n{r.stdout}{r.stderr}")

for p in problems:
    print("examples: " + p)
for p in sorted(pending):
    print("examples: pending " + p)
print(f"examples: {ran} blocks and files checked, {len(problems)} problems, {len(pending)} pending by ticket")
sys.exit(1 if problems else 0)
