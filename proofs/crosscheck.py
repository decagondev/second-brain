"""Explicit-state cross-check of the design logic in Approvals.tla, Rollout.tla and Handoff.tla.
Not a substitute for TLC: it mirrors the specs by hand. Used to sanity-check the
invariants and the mutation claims before SB-011/SB-012/SB-705 run the real thing."""
from collections import deque
from itertools import product

def bfs(init, nxt, invs, act_props=()):
    seen, q = set(init), deque(init)
    while q:
        s = q.popleft()
        for name, f in invs:
            if not f(s): return f"VIOLATION {name}", len(seen)
        for t in nxt(s):
            for name, p in act_props:
                if not p(s, t): return f"VIOLATION {name}", len(seen)
            if t not in seen: seen.add(t); q.append(t)
    return "ok", len(seen)

# ---------------- Approvals
U, P, MAXE = ("u1", "u2"), ("p1", "p2"), 1
def approvals(mut=None):
    # state: (st, req, pay, appr_hash, approver, killed, audit(tuple), receipt, edits)
    init = [("Drafted", r, p, None, None, False, (), False, 0) for r in U for p in P]
    def nxt(s):
        st, req, pay, ah, ap, k, au, rc, e = s
        out = []
        if st == "Drafted": out.append(("PendingApproval", req, pay, ah, ap, k, au + (("PendingApproval", pay),), rc, e))
        if st in ("Drafted", "PendingApproval", "Approved") and e < MAXE:
            for p in P:
                if p != pay: out.append(("Drafted", req, p, None, None, k, au + (("Drafted", None),), rc, e + 1))
        if st == "Approved":
            for p in P:
                if p != pay: out.append((st, req, p, ah, ap, k, au, rc, e))          # Tamper
        if st == "PendingApproval":
            for u in U:
                if mut == "selfapprove" or u != req:
                    out.append(("Approved", req, pay, pay, u, k, au + (("Approved", pay),), rc, e))
            out.append(("Rejected", req, pay, ah, ap, k, au + (("Rejected", pay),), rc, e))
        if st in ("PendingApproval", "Approved"): out.append(("Expired", req, pay, ah, ap, k, au + (("Expired", None),), rc, e))
        if st == "Approved" and (mut == "nokill" or not k) and (mut == "nohash" or pay == ah):
            out.append(("Executing", req, pay, ah, ap, k, au if mut == "nolog" else au + (("Executing", pay),), rc, e))
        if st == "Executing":
            out.append(("Ran", req, pay, ah, ap, k, au + (("Ran", pay),), mut != "noreceipt", e))
            out.append(("DidNotRun", req, pay, ah, ap, k, au + (("DidNotRun", pay),), rc, e))
        if not k: out.append((st, req, pay, ah, ap, True, au + (("Kill", None),), rc, e))
        return out
    X = ("Executing", "Ran", "DidNotRun")
    invs = [
        ("AuditBeforeExec", lambda s: s[0] not in X or ("Executing", s[3]) in s[6]),
        ("ApprovalBeforeExec", lambda s: s[0] not in X or (s[3] is not None and s[4] is not None)),
        ("ExecutedMatchesApproved", lambda s: s[0] not in X or s[2] == s[3]),
        ("NoSelfApproval", lambda s: s[4] != s[1]),
        ("RanImpliesReceipt", lambda s: s[0] != "Ran" or s[7]),
        ("AtMostOnce", lambda s: sum(1 for r in s[6] if r[0] == "Executing") <= 1),
    ]
    props = [("KillWins", lambda s, t: not s[5] or t[0] != "Executing" or s[0] == "Executing")]
    return bfs(init, nxt, invs, props)

# ---------------- Rollout (non-atomic rollback, BUG-0001)
ST, TT, MAXR = ("s1", "s2"), ("t1",), 4
TEN = ST + TT
def rollout(mut=None):
    # state: (latest, gated, revoked, stable, prev, running(tuple per TEN), draining(frozenset of indexes))
    init = [(0, frozenset({0}), frozenset(), 0, 0, tuple(0 for _ in TEN), frozenset())]
    def nxt(s):
        lat, g, rv, sb, pv, run, dr = s
        out = []
        if lat < MAXR:
            out.append((lat + 1, g | {lat + 1}, rv, sb, pv, run, dr)); out.append((lat + 1, g, rv, sb, pv, run, dr))
        for i, t in enumerate(TEN):
            if t in TT and i not in dr:
                for r in range(MAXR + 1):
                    if r in g and (r not in rv or mut == "deployrevoked") and r >= sb:
                        out.append((lat, g, rv, sb, pv, run[:i] + (r,) + run[i+1:], dr))
            if t in ST and i not in dr and run[i] != sb:
                out.append((lat, g, rv, sb, pv, run[:i] + (sb,) + run[i+1:], dr))
            if i in dr:
                out.append((lat, g, rv, sb, pv, run[:i] + (sb,) + run[i+1:], dr - {i}))
        for r in range(MAXR + 1):
            okg = (mut == "nogate" and r <= lat) or r in g
            guard = mut == "noguard" or all(run[i] == sb for i, t in enumerate(TEN) if t in ST)
            if okg and r not in rv and r > sb and guard:
                out.append((lat, g, rv, r, sb, run, dr))
            if r in g and r not in rv and r > sb:
                out.append((lat, g, rv | {r}, sb, pv, run, dr | {i for i, x in enumerate(run) if x == r}))
        if pv < sb:
            moved = {i for i, x in enumerate(run) if x == sb}
            out.append((lat, g, rv | {sb}, pv, pv, run, dr if mut == "nodrain" else dr | moved))
        return out
    serving = lambda s: [i for i, t in enumerate(TEN) if t in ST and i not in s[6]]
    invs = [
        ("OnlyGatedRuns", lambda s: all(x in s[1] for x in s[5])),
        ("RevokedOnlyWhileDraining", lambda s: all(i in s[6] for i, x in enumerate(s[5]) if x in s[2])),
        ("DrainingOnlyRevoked", lambda s: all(s[5][i] in s[2] for i in s[6])),
        ("OneBehindMax", lambda s: all(s[5][i] in (s[3], s[4]) for i in serving(s))),
        ("StableNotAhead", lambda s: all(s[5][i] <= s[3] for i in serving(s))),
        ("StableNotRevoked", lambda s: s[3] not in s[2]),
    ]
    return bfs(init, nxt, invs)

# ---------------- Handoff
PATHS, RETAINED, PLANS = frozenset({"deploy", "breakglass", "iam", "publish"}), frozenset({"publish"}), ("h1", "h2")
INFLIGHT = ("Planned", "Provisioned", "KeyDual", "Moved", "Verified")
def handoff(mut=None):
    # state: (phase, plan, approvals, wraps, access, serving)
    init = [("Managed", None, frozenset(), frozenset({"factory"}), PATHS, "factoryCell")]
    def nxt(s):
        ph, pl, ap, wr, ac, sv = s
        both = all((p, pl) in ap for p in ("factory", "tenant"))
        out = []
        if ph in ("Managed", "Planned"):
            for h in PLANS: out.append(("Planned", h, frozenset() if mut != "keepapprovals" else ap, wr, ac, sv))
        if ph == "Planned":
            for p in ("factory", "tenant"): out.append((ph, pl, ap | {(p, pl)}, wr, ac, sv))
            if both or mut == "nodual": out.append(("Provisioned", pl, ap, wr, ac, sv))
        if ph == "Provisioned": out.append(("KeyDual", pl, ap, wr | {"tenant"}, ac, sv))
        if ph == "KeyDual": out.append(("Moved", pl, ap, wr, ac, sv))
        if ph == "Moved": out.append(("Verified", pl, ap, wr, ac, sv))
        if ph == "Verified" or (mut == "cutbeforewrap" and ph == "Provisioned"):
            out.append(("CutOver", pl, ap, wr, ac if mut == "norevoke" else RETAINED, "tenantCell"))
        if ph == "CutOver" or (mut == "destroyfirst" and ph == "Provisioned"):
            out.append(("TenantOwned" if ph == "CutOver" else ph, pl, ap, wr - {"factory"}, ac, sv))
        if ph in INFLIGHT: out.append(("Managed", None, frozenset(), frozenset({"factory"}), ac, sv))
        if ph == "TenantOwned": out.append(("Detached", pl, ap, wr, frozenset(), sv))
        return out
    invs = [
        ("KeyNeverOrphaned", lambda s: len(s[3]) > 0),
        ("ServingCanUnwrap", lambda s: ("factory" if s[5] == "factoryCell" else "tenant") in s[3]),
        ("NoUnlistedAccess", lambda s: (s[0] not in ("CutOver", "TenantOwned") or s[4] <= RETAINED) and (s[0] != "Detached" or not s[4])),
        ("FactoryKeyGone", lambda s: s[0] not in ("TenantOwned", "Detached") or "factory" not in s[3]),
        ("DualApproval", lambda s: s[0] in ("Managed", "Planned") or all((p, s[1]) in s[2] for p in ("factory", "tenant"))),
        ("ManagedIsIntact", lambda s: s[0] != "Managed" or (s[5] == "factoryCell" and s[3] == {"factory"} and s[4] == PATHS)),
    ]
    return bfs(init, nxt, invs)

print("Approvals            ", approvals())
for m in ("selfapprove", "nohash", "nokill", "nolog", "noreceipt"):
    print(f"Approvals mutant {m:11}", approvals(m))
print("Rollout              ", rollout())
for m in ("noguard", "nogate", "deployrevoked", "nodrain"):
    print(f"Rollout mutant {m:13}", rollout(m))
print("Handoff              ", handoff())
for m in ("nodual", "destroyfirst", "norevoke", "cutbeforewrap"):
    print(f"Handoff mutant {m:13}", handoff(m))
