"""python3 -m evals.pilot report EXPORT.jsonl: the pilot's Gate 1 numbers from the
weekly feedback export (sb-onboarding(7), Running the pilot review; SB-311).

The export holds ids, counts and comments only (sb-ask(3)); so does this report."""
import json
import sys
from collections import Counter

THUMBS_UP_MIN = 0.70


def summarise(lines):
    rows = [json.loads(l) for l in lines if l.strip()]
    n = len(rows)
    up = sum(r.get("rating") == "up" for r in rows)
    wrong = Counter(r["wrong_citation"] for r in rows if r.get("wrong_citation"))
    with_uncited = sum(r.get("uncited", 0) > 0 for r in rows)
    with_refused = sum(r.get("refused", 0) > 0 for r in rows)
    return {
        "feedback": n,
        "thumbs_up_rate": up / n if n else 0.0,
        "wrong_citations": dict(sorted(wrong.items(), key=lambda kv: (-kv[1], kv[0]))),
        "answers_with_uncited": with_uncited / n if n else 0.0,
        "answers_with_refused": with_refused / n if n else 0.0,
        "releases": sorted({r.get("release", "") for r in rows}),
    }


def passes(s):
    return s["feedback"] > 0 and s["thumbs_up_rate"] >= THUMBS_UP_MIN


def main(argv):
    if len(argv) != 2 or argv[0] != "report":
        print("usage: python3 -m evals.pilot report EXPORT.jsonl", file=sys.stderr)
        return 2
    with open(argv[1], encoding="utf-8") as f:
        s = summarise(f)
    print(f"feedback              {s['feedback']}")
    print(f"thumbs-up rate        {s['thumbs_up_rate']:.1%} (gate {THUMBS_UP_MIN:.0%})")
    print(f"answers with uncited  {s['answers_with_uncited']:.1%}")
    print(f"answers with refused  {s['answers_with_refused']:.1%}")
    print(f"releases              {', '.join(s['releases']) or '-'}")
    for cite, k in s["wrong_citations"].items():
        print(f"wrong citation        {cite}  x{k}")
    if not passes(s):
        print("pilot: Gate 1 thumbs-up rate not met (or no feedback) [EEVAL]", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
