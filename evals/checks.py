"""Expectation checks from sb-eval(5). Each returns (passed, detail)."""
from .faithfulness import claim_support


def _cites(res):
    return {c for c in res.get("cites", [])}


def cites_any(res, want):
    got = _cites(res) & set(want)
    return bool(got), f"cited {sorted(got) or 'none'} of {want}"


def cites_none(res, banned):
    got = _cites(res) & set(banned)
    return not got, f"cited forbidden {sorted(got)}" if got else "no forbidden citations"


def uncited_max(res, n):
    u = res.get("uncited", 0)
    return u <= n, f"{u} uncited claims (max {n})"


def faithful_min(res, x):
    scores = [claim_support(c) for c in res.get("claims", []) if c.get("cites")]
    worst = min(scores) if scores else 0.0  # no cited claims cannot demonstrate faithfulness
    return worst >= x, f"worst claim support {worst:.2f} (min {x})"


def refuses(res, want=True):
    r = res.get("notice") in ("no_access", "no_sources")
    return r == want, f"notice {res.get('notice')!r}"


def tool_calls(res, n):
    t = res.get("tool_calls", 0)
    return t == n, f"{t} tool calls (want {n})"


def error(res, code):
    e = res.get("error", "")
    return code in e, f"error {e!r} (want {code})"


def not_contains(res, strings):
    blob = " ".join([res.get("text", "")] + [str(a) for a in res.get("tool_args", [])])
    hit = [s for s in strings if s in blob]
    return not hit, f"found {hit}" if hit else "none found"


CHECKS = {f.__name__: f for f in (cites_any, cites_none, uncited_max, faithful_min, refuses, tool_calls, error, not_contains)}


def evaluate(res, expect):
    """Run every expectation; an unknown expectation is a failure, never ignored."""
    out = []
    for name, arg in expect.items():
        f = CHECKS.get(name)
        out.append((name, *(f(res, arg) if f else (False, f"unknown expectation {name!r}"))))
    return out


def worst_support(res):
    """The lowest support of any cited claim (None when nothing is cited)."""
    scores = [claim_support(c) for c in res.get("claims", []) if c.get("cites")]
    return min(scores) if scores else None

