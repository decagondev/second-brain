#!/usr/bin/env python3
"""SB-1404: the manual and the code agree. Every HTTP endpoint the cell serves is in the
manual, every endpoint the manual describes is served (or listed below as not built yet,
with its ticket), every two-word `sb` command the CLI dispatches is in sb(1)'s synopsis
and the other way round, and every flag the CLI parses is in a man page. Error codes are
generated from sb-errors(7) and checked by `make gen-check`. A self-test runs first."""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

# Endpoints the manual specifies that no cell serves yet, with the ticket that builds them.
# The check fails once one is built and still listed here.
NOT_BUILT = {}

ROUTE = re.compile(r'Handle(?:Func)?\("(GET|POST|PUT|PATCH|DELETE) (/[^"]*)"')
MANUAL_ROUTE = re.compile(
    r'\b(GET|POST|PUT|PATCH|DELETE)\s+(/(?:v1|scim|metrics|verify|healthz|login|logout|callback|chat|Users|Groups)[^\s`\'")\]|,;]*|/(?=[\s`]))')
CASE = re.compile(r'case "([a-z][a-z-]*(?: [a-z][a-z-]*)?)"')
PLANNED = re.compile(r'"([a-z]+ [a-z-]+)":\s*"SB-\d+"')
FLAG = re.compile(r'"(--[a-z][a-z-]+)"')


def norm(path):
    path = path.split("?")[0].rstrip(".:")
    if path.startswith(("/Users", "/Groups")):
        path = "/scim/v2" + path  # sb-identity(7) writes SCIM paths relative to its base
    parts = []
    for p in path.split("/"):
        if p.startswith("{") or re.fullmatch(r"[A-Z][A-Z_]*", p) or p in ("…", "..."):
            parts.append("{}")
        else:
            parts.append(p)
    return "/".join(parts)


def code_routes(texts):
    return {(m, norm(p)) for t in texts for m, p in ROUTE.findall(t)}


def manual_routes(texts):
    return {(m, norm(p)) for t in texts for m, p in MANUAL_ROUTE.findall(t)}


def route_problems(code, manual, not_built):
    probs = [f"{m} {p} is served but not in the manual" for m, p in sorted(code - manual)]
    for m, p in sorted(manual - code):
        if f"{m} {p}" not in not_built:
            probs.append(f"{m} {p} is in the manual but not served (build it, or list it in NOT_BUILT with its ticket)")
    for k in sorted(not_built):
        m, p = k.split(" ", 1)
        if (m, p) in code:
            probs.append(f"{k} is built now: take it off NOT_BUILT")
    return probs


def synopsis_commands(sb1, families):
    syn = sb1[sb1.index("## SYNOPSIS"):sb1.index("## DESCRIPTION")]
    out = set()
    for line in syn.splitlines():
        m = re.match(r"\s*sb\s+([a-z-]+)\s+([a-z|-]+)", line)
        if m and m.group(1) in families:  # one-word commands take their own arguments
            out |= {m.group(1) + " " + sub for sub in m.group(2).split("|")}
    return out


def self_test():
    code = code_routes(['mux.HandleFunc("GET /v1/spaces/{name}/reviews", h.x)', 'mux.Handle("GET /metrics", m)'])
    man = manual_routes(["`GET /v1/spaces/NAME/reviews` lists them; POST /v1/holds/ID/release too; `POST /Users` creates."])
    assert code == {("GET", "/v1/spaces/{}/reviews"), ("GET", "/metrics")}, code
    assert ("POST", "/v1/holds/{}/release") in man and ("POST", "/scim/v2/Users") in man, man
    probs = route_problems(code, man, {})
    assert "GET /metrics is served but not in the manual" in probs, probs
    assert route_problems(code, code, {"GET /metrics": "SB-0"}) == ["GET /metrics is built now: take it off NOT_BUILT"]
    syn = "## SYNOPSIS\n```\nsb tenant  check|apply FILE\nsb kill agent|tenant\n```\n## DESCRIPTION"
    assert synopsis_commands(syn, {"tenant"}) == {"tenant check", "tenant apply"}


def main():
    self_test()
    # Fakes of outside services (internal/adapters/**/xxxtest) serve those services' APIs.
    go = [p.read_text(encoding="utf-8") for p in ROOT.glob("internal/**/*.go")
          if not p.name.endswith("_test.go") and not p.parent.name.endswith("test")]
    go += [p.read_text(encoding="utf-8") for p in ROOT.glob("cmd/**/*.go") if not p.name.endswith("_test.go")]
    man = [p.read_text(encoding="utf-8") for p in (ROOT / "manual").glob("*.md")]
    routes = code_routes(go)
    probs = route_problems(routes, manual_routes(man), NOT_BUILT)

    cli = (ROOT / "internal" / "cli" / "cli.go").read_text(encoding="utf-8")
    cmds = {c for c in CASE.findall(cli[cli.index("switch"):]) if " " in c} | set(PLANNED.findall(cli))
    # One-word cases that pick a subcommand themselves: case "eval": if rest[1] == "run" ...
    for fam, block in re.findall(r'case "([a-z]+)":\n((?:\t\t.*\n)+)', cli):
        cmds |= {fam + " " + sub for sub in re.findall(r'rest\[1\] == "([a-z-]+)"', block)}
    listed = synopsis_commands((ROOT / "manual" / "sb.1.md").read_text(encoding="utf-8"), {c.split(" ")[0] for c in cmds})
    probs += [f"`sb {c}` is a command but not in sb(1)'s synopsis" for c in sorted(cmds - listed)]
    probs += [f"`sb {c}` is in sb(1)'s synopsis but isn't a command" for c in sorted(listed - cmds)]

    flags = set()
    for p in (ROOT / "internal" / "cli").glob("*.go"):
        if not p.name.endswith("_test.go"):
            flags |= set(FLAG.findall(p.read_text(encoding="utf-8")))
    alltext = "\n".join(man)
    probs += [f"flag {f} is parsed but not in any man page" for f in sorted(flags) if f not in alltext]

    for p in probs:
        print(f"manual-code: {p}")
    if probs:
        sys.exit(1)
    print(f"manual-code: {len(routes)} endpoints, {len(cmds)} commands and {len(flags)} flags match the manual")


main()
