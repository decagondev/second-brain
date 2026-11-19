#!/usr/bin/env python3
"""SB-008: enforce the dependency rule from docs/architecture/00-overview.md.

| Package may import  | core/ports | core/* | adapters/* | vendor |
| internal/core/ports | -          | no     | no         | no     |
| internal/core/*     | yes        | yes    | no         | no     |
| internal/adapters/* | yes        | yes    | no (sibs)  | yes    |
| cmd/*               | yes        | yes    | yes        | yes    |

Runs `go list -json ./...` and checks every import. Exit 1 on any violation.

Also enforces the module allow-list (ADR-0017): every go.sum line must appear in
tools/deps/allowed.txt, and every module required by go.mod must be allowed at that
exact version.
"""
import json, subprocess, sys

MOD = "example.com/second-brain"
CORE, PORTS, ADAPT = f"{MOD}/internal/core", f"{MOD}/internal/core/ports", f"{MOD}/internal/adapters"
TENANCY = f"{MOD}/internal/core/tenancy"

def std(p):
    return "." not in p.split("/")[0]

def under(p, root):
    return p == root or p.startswith(root + "/")

def adapter_of(p):
    parts = p[len(ADAPT) + 1:].split("/")
    return "/".join(parts[:2]) if parts and parts[0] == "connectors" else parts[0]

def violations(pkgs):
    bad = []
    for pkg in pkgs:
        p = pkg["ImportPath"]
        for imp in sorted(set(pkg.get("Imports", []) + pkg.get("TestImports", []))):
            if under(p, PORTS):
                if not (std(imp) or imp == TENANCY):
                    bad.append((p, imp, "core/ports may import only the standard library and core/tenancy"))
            elif under(p, CORE):
                if not (std(imp) or under(imp, CORE)):
                    bad.append((p, imp, "core may import only the standard library and core"))
            elif under(p, ADAPT) and under(imp, ADAPT) and adapter_of(imp) != adapter_of(p):
                bad.append((p, imp, "adapters may not import sibling adapters"))
    return bad

def issuer_problems(root):
    """SB-108: only the auth middleware may hold a tenancy.Issuer (tests may too)."""
    import re
    from pathlib import Path
    bad = []
    for f in Path(root).rglob("*.go"):
        rel = f.relative_to(root).as_posix()
        # laws/ and tools/evalcell/ are test harnesses that stand in for sign-in and never
        # ship (SB-010, SB-309; the tools/evalcell/ exemption approved by the user 2026-11-19).
        if rel.startswith(("vendor/", "internal/auth/", "internal/core/tenancy/", "laws/", "tools/evalcell/")) or rel.endswith("_test.go"):
            continue
        if re.search(r"\btenancy\.NewIssuer\(",f.read_text(encoding="utf-8")):
            bad.append(f"{rel} calls tenancy.NewIssuer; only internal/auth may (SB-108)")
    return bad

def allowlist_problems(root):
    from pathlib import Path
    root = Path(root)
    allowed = {l.strip() for l in (root / "tools/deps/allowed.txt").read_text().splitlines()
               if l.strip() and not l.startswith("#")}
    allowed_mods = {tuple(l.split()[:2]) for l in allowed if not l.split()[1].endswith("/go.mod")}
    bad = []
    gosum = root / "go.sum"
    for l in (gosum.read_text().splitlines() if gosum.exists() else []):
        if l.strip() and l.strip() not in allowed:
            bad.append(f"go.sum line not in tools/deps/allowed.txt (ADR-0017): {l.strip()}")
    in_block = False
    for l in (root / "go.mod").read_text().splitlines():
        t = l.split("//")[0].strip()
        if t.startswith("require ("):
            in_block = True
            continue
        if in_block and t == ")":
            in_block = False
            continue
        if t.startswith("require "):
            t = t[len("require "):].strip()
        elif not in_block:
            continue
        if not t:
            continue
        mod, ver = t.split()[:2]
        if (mod, ver) not in allowed_mods:
            bad.append(f"go.mod requires {mod} {ver}, which tools/deps/allowed.txt does not allow (ADR-0017)")
    return bad

if __name__ == "__main__":
    out = subprocess.run(["go", "list", "-json", "./..."], capture_output=True, text=True)
    if out.returncode != 0:
        sys.exit("deps.py: go list failed:\n" + out.stderr)
    dec, s, pkgs = json.JSONDecoder(), out.stdout, []
    i = 0
    while i < len(s):
        while i < len(s) and s[i].isspace():
            i += 1
        if i >= len(s):
            break
        obj, i = dec.raw_decode(s, i)
        pkgs.append(obj)
    bad = violations(pkgs)
    for p, imp, why in bad:
        print(f"deps: {p} imports {imp}: {why}")
    from pathlib import Path
    mods = allowlist_problems(Path(__file__).resolve().parents[2]) + issuer_problems(Path(__file__).resolve().parents[2])
    for m in mods:
        print("deps: " + m)
    print(f"deps: {len(pkgs)} packages checked, {len(bad)} violations; module allow-list and issuer rule: {len(mods)} problems")
    sys.exit(1 if bad or mods else 0)
