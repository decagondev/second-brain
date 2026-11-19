#!/usr/bin/env python3
"""SB-112: policy analysis (sb-policy(5), Rules CI enforces).

  python3 tools/check/policy.py              validate every policy and check tenant scope
  python3 tools/check/policy.py --diff REF   also print the permission diff against git REF

1. Every .cedar file under policy/ and examples/policies/ validates against
   policy/schema.cedarschema (cedarpy). A missing validator fails; it never skips.
2. Every `permit` carries `principal.tenant == resource.tenant` in its `when` (LAW-02).
3. Tenant policies (examples/policies/) may `permit` only use_agent, propose and approve.
4. --diff evaluates a fixed matrix of principals, actions and resources against the old
   and new policy sets and prints each permission gained (+) or lost (-).
A self-test runs first, so a rule can't silently stop matching (see BUG-0005).
"""
import re
import subprocess
import sys
from pathlib import Path

try:
    import cedarpy
except ImportError as e:  # unknown means no
    raise SystemExit("policy.py needs cedarpy: pip install cedarpy") from e

ROOT = Path(__file__).resolve().parents[2]
SCHEMA_PATH = "policy/schema.cedarschema"
TENANT_COND = re.compile(r"principal\s*\.\s*tenant\s*==\s*resource\s*\.\s*tenant")
TENANT_PERMITS = {"use_agent", "propose", "approve"}


def strip_comments(text):
    return re.sub(r"//[^\n]*", "", text)


def statements(text):
    """Split Cedar text into policy statements (top-level ';', ignoring strings)."""
    out, cur, in_str = [], [], False
    for ch in strip_comments(text):
        cur.append(ch)
        if ch == '"':
            in_str = not in_str
        elif ch == ";" and not in_str:
            s = "".join(cur).strip()
            if s:
                out.append(s)
            cur = []
    return out


def scope_problems(name, text, tenant_policy):
    probs = []
    for st in statements(text):
        if not st.startswith("permit"):
            continue
        head = st.split(")", 1)[0]
        when = st[len(head):]
        if not re.search(r"\bwhen\b", when) or not TENANT_COND.search(when):
            probs.append(f"{name}: permit without `principal.tenant == resource.tenant` in its when: {' '.join(st.split())[:90]}")
        if tenant_policy:
            m = re.search(r'action\s*==\s*Action::"([a-z_]+)"', head)
            if not m or m.group(1) not in TENANT_PERMITS:
                probs.append(f"{name}: tenant policies may permit only {sorted(TENANT_PERMITS)}: {' '.join(st.split())[:90]}")
    return probs


def policy_files(root):
    files = {}
    for glob in ("policy/**/*.cedar", "examples/policies/*.cedar"):
        for f in sorted(root.glob(glob)):
            files[f.relative_to(root).as_posix()] = f.read_text()
    return files


def check(files, schema):
    probs = []
    for name, text in files.items():
        r = cedarpy.validate_policies(text, schema)
        if not r.validation_passed:
            probs.append(f"{name}: does not validate against the schema: {[str(e) for e in r.errors]}")
        probs += scope_problems(name, text, name.startswith("examples/policies/"))
    return probs


# The permission matrix for --diff: who, doing what, to which resource.
GROUPS = ["grp_it_leads", "grp_contractors", "grp_finance"]
RESOURCES = {
    "read": [('Chunk::"c"', {"tenant": {"__entity": {"type": "Tenant", "id": "t"}}, "cite": "docs:c@1"}, "Chunk"),
             ('Document::"d"', {"tenant": {"__entity": {"type": "Tenant", "id": "t"}}, "source": "docs"}, "Document")],
    "use_agent": [(f'Agent::"{a}"', {"tenant": {"__entity": {"type": "Tenant", "id": "t"}}, "name": a, "consequential": False}, "Agent")
                  for a in ("ask", "draft-reply")],
    "approve": [('ProposedAction::"p"', {"tenant": {"__entity": {"type": "Tenant", "id": "t"}}, "workflow": "ticket-update",
                                          "requester": {"__entity": {"type": "User", "id": "other"}}}, "ProposedAction")],
    "admin": [('Cell::"t"', {"tenant": {"__entity": {"type": "Tenant", "id": "t"}}}, "Cell")],
}


def matrix(policies_text):
    allowed = set()
    for groups in [[]] + [[g] for g in GROUPS]:
        who = "+".join(groups) or "member"
        for action, resources in RESOURCES.items():
            for uid, attrs, typ in resources:
                rid = uid.split("::")[1].strip('"')
                ents = [{"uid": {"type": "Tenant", "id": "t"}, "attrs": {}, "parents": []},
                        {"uid": {"type": "User", "id": "u"}, "attrs": {"tenant": {"__entity": {"type": "Tenant", "id": "t"}}},
                         "parents": [{"type": "Group", "id": g} for g in groups]},
                        {"uid": {"type": typ, "id": rid}, "attrs": attrs, "parents": []}]
                ents += [{"uid": {"type": "Group", "id": g}, "attrs": {"tenant": {"__entity": {"type": "Tenant", "id": "t"}}}, "parents": []} for g in groups]
                req = {"principal": 'User::"u"', "action": f'Action::"{action}"', "resource": uid, "context": {}}
                if cedarpy.is_authorized(req, policies_text, ents).allowed:
                    allowed.add(f"{who} may {action} {uid}")
    return allowed


def git_files(ref):
    names = subprocess.run(["git", "ls-tree", "-r", "--name-only", ref, "policy", "examples/policies"],
                           cwd=ROOT, capture_output=True, text=True, check=True).stdout.split()
    return {n: subprocess.run(["git", "show", f"{ref}:{n}"], cwd=ROOT, capture_output=True, text=True, check=True).stdout
            for n in names if n.endswith(".cedar")}


def diff(old, new):
    a, b = matrix("\n".join(old.values())), matrix("\n".join(new.values()))
    return [f"+ {p}" for p in sorted(b - a)] + [f"- {p}" for p in sorted(a - b)]


def self_test(schema):
    bad = 'permit (principal, action == Action::"read", resource is Chunk);\n'
    good = 'permit (principal, action == Action::"read", resource is Chunk) when { principal.tenant == resource.tenant };\n'
    assert len(scope_problems("x", bad, False)) == 1, "policy.py self-test: scope rule does not match"
    assert scope_problems("x", good, False) == [], "policy.py self-test: scope rule matches good policy"
    assert scope_problems("x", good, True), "policy.py self-test: tenant read permit not refused"
    assert diff({"a": ""}, {"a": good}), "policy.py self-test: diff sees no change"


if __name__ == "__main__":
    schema = (ROOT / SCHEMA_PATH).read_text()
    self_test(schema)
    files = policy_files(ROOT)
    probs = check(files, schema)
    for p in probs:
        print("policy: " + p)
    print(f"policy: {len(files)} files checked, {len(probs)} problems")
    if "--diff" in sys.argv:
        ref = sys.argv[sys.argv.index("--diff") + 1]
        lines = diff(git_files(ref), files)
        print(f"policy: permission diff against {ref}:")
        print("\n".join("  " + l for l in lines) if lines else "  (no change)")
    sys.exit(1 if probs else 0)
