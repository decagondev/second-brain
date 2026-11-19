#!/usr/bin/env python3
"""SB-011..013: mutation checks for the formal models (table in proofs/README.md).

Each mutation weakens one guard; the checker must report the expected counterexample.
Run after `make tools`: python3 tools/ci/mutations.py (exits 1 if any mutation is missed).
"""
import os, re, shutil, subprocess, sys, tempfile
R = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
TLA = os.path.join(R, "tools", "vendor", "tla2tools.jar")
ALLOY = os.path.join(R, "tools", "vendor", "alloy.jar")

M = [  # (model, old, new, expected)
 ("Approvals", "  /\\ u # requester\n", "", "NoSelfApproval"),
 ("Approvals", "  /\\ ~killed\n  /\\ payload = approvedHash\n", "  /\\ ~killed\n", "AuditBeforeExec|ExecutedMatchesApproved"),
 ("Approvals", "  /\\ ~killed\n  /\\ payload = approvedHash\n", "  /\\ payload = approvedHash\n", "KillWins"),
 ("Approvals", "  /\\ state' = \"Executing\"\n  /\\ Log(\"Executing\", payload)\n", "  /\\ state' = \"Executing\"\n  /\\ audit' = audit\n", "AuditBeforeExec"),
 ("Rollout", "  /\\ \\A t \\in StableTenants : running[t] = stable\n", "", "OneBehindMax"),
 ("Rollout", "Promote(r) ==\n  /\\ r \\in gated \\ revoked\n", "Promote(r) ==\n  /\\ r \\notin revoked\n", "OnlyGatedRuns"),
 ("Rollout", "  /\\ r \\in gated \\ revoked\n  /\\ r >= stable\n", "  /\\ r \\in gated\n  /\\ r >= stable\n", "RevokedOnlyWhileDraining"),
 ("Rollout", "  /\\ draining' = draining \\cup {t \\in Tenants : running[t] = stable}\n  /\\ UNCHANGED <<latest, gated, prevStable, running>>",
             "  /\\ UNCHANGED <<latest, gated, prevStable, running, draining>>", "RevokedOnlyWhileDraining"),
 ("Handoff", "  /\\ BothApproved\n", "", "DualApproval"),
 ("Handoff", "DestroyFactoryWrap ==\n  /\\ phase = \"CutOver\"\n", "DestroyFactoryWrap ==\n  /\\ phase \\in {\"Provisioned\", \"CutOver\"}\n", "KeyNeverOrphaned"),
 ("Handoff", "  /\\ access' = Retained\n", "  /\\ access' = access\n", "NoUnlistedAccess"),
 ("Handoff", "Cutover ==\n  /\\ phase = \"Verified\"\n", "Cutover ==\n  /\\ phase \\in {\"Provisioned\", \"Verified\"}\n", "ServingCanUnwrap"),
]

def tlc(name, text):
    d = tempfile.mkdtemp()
    open(os.path.join(d, name + ".tla"), "w", newline="\n").write(text)
    shutil.copy(os.path.join(R, "proofs", "tla", name + ".cfg"), d)
    out = subprocess.run(["java", "-XX:+UseParallelGC", "-cp", TLA, "tlc2.TLC", "-workers", "auto", "-deadlock",
                          "-config", name + ".cfg", name + ".tla"], cwd=d, capture_output=True, text=True).stdout
    shutil.rmtree(d, ignore_errors=True)
    m = re.search(r"(Invariant|Temporal property|Action property) (\w+) is violated", out) or re.search(r"Error: (.*) is violated", out)
    if m: return m.group(2) if m.lastindex >= 2 else m.group(1)
    if "No error has been found" in out: return "NO ERROR"
    return "OTHER: " + out[-400:]

ok = 0
for i, (name, old, new, exp) in enumerate(M, 1):
    src = open(os.path.join(R, "proofs", "tla", name + ".tla")).read()
    assert src.count(old) == 1, (i, "pattern not unique/found")
    got = tlc(name, src.replace(old, new))
    hit = re.fullmatch(exp, got) is not None
    ok += hit
    print(f"{i:2} {name:9} expected {exp:40} got {got:28} {'CAUGHT' if hit else 'MISSED'}")

# Alloy mutation
als = open(os.path.join(R, "proofs", "alloy", "Permissions.als")).read()
old = "a.context = { d: a.candidates | a.live[d] = Allow }"
assert als.count(old) == 1
d = tempfile.mkdtemp(); p = os.path.join(d, "Permissions.als")
open(p, "w", newline="\n").write(als.replace(old, "a.context = a.candidates - a.live.Deny"))
res = {}
for c in ["NoCrossTenant", "CitedImpliesLiveAllow", "UnknownNeverUsed", "MissingNeverUsed", "StalePrefilterHarmless"]:
    out = subprocess.run(["java", "-jar", ALLOY, "exec", "-f", "-c", c, "-t", "none", "-o", os.path.join(d, c), p],
                         stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True).stdout
    res[c] = "UNSAT" if re.search(r"check\s+%s\s+\d+\s+UNSAT" % c, out) else ("SAT" if "SAT" in out else "OTHER")
hit = res["UnknownNeverUsed"] == "SAT" and res["MissingNeverUsed"] == "SAT"
ok += hit
print(f"13 Permissions context = candidates - Deny: {res} {'CAUGHT' if hit else 'MISSED'}")
print(f"caught {ok} of {len(M) + 1}")
sys.exit(0 if ok == len(M) + 1 else 1)
