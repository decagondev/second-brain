#!/usr/bin/env python3
"""SB-1405: every package under internal/core is covered by the unit, law and suite tests
together above a floor. It reads the profiles `make test`, `make laws` and `make smoke`
write to build/cover, merges them (a statement counts once, covered if any run covered
it), prints a figure per package, and fails if one is below the floor. Test helpers
(packages named *test, and connectorkit) are reported but have no floor. A self-test
runs first."""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
FLOOR = 75.0
HELPERS = re.compile(r"(test|connectorkit)$")
LINE = re.compile(r"^(\S+):(\d+\.\d+,\d+\.\d+) (\d+) (\d+)$")


def merge(texts):
    blocks = {}
    for t in texts:
        for line in t.splitlines():
            m = LINE.match(line)
            if not m:
                continue
            key = (m.group(1), m.group(2))
            n, c = int(m.group(3)), int(m.group(4))
            old = blocks.get(key, (n, 0))
            blocks[key] = (n, max(old[1], c))
    total, covered = {}, {}
    for (f, _), (n, c) in blocks.items():
        pkg = f.rsplit("/", 1)[0].replace("example.com/second-brain/", "")
        total[pkg] = total.get(pkg, 0) + n
        covered[pkg] = covered.get(pkg, 0) + (n if c > 0 else 0)
    return {p: 100.0 * covered[p] / total[p] for p in total if total[p]}


def below(figures, floor):
    return sorted(p for p, v in figures.items() if v < floor and not HELPERS.search(p))


def self_test():
    a = "mode: set\nexample.com/second-brain/internal/core/x/x.go:1.1,2.2 3 1\nexample.com/second-brain/internal/core/x/x.go:3.1,4.2 1 0\n"
    b = "mode: set\nexample.com/second-brain/internal/core/x/x.go:3.1,4.2 1 1\nexample.com/second-brain/internal/core/xtest/t.go:1.1,2.2 5 0\n"
    f = merge([a])
    assert abs(f["internal/core/x"] - 75.0) < 1e-9, f
    f = merge([a, b])
    assert f["internal/core/x"] == 100.0 and f["internal/core/xtest"] == 0.0, f
    assert below(f, 70) == [], "a helper package must have no floor"
    assert below({"internal/core/y": 50.0}, 70) == ["internal/core/y"]


def main():
    self_test()
    outs = sorted((ROOT / "build" / "cover").glob("*.out"))
    if not outs:
        print("coverage: no profiles in build/cover; run make test, make laws and make smoke first")
        sys.exit(1)
    figures = merge(p.read_text(encoding="utf-8") for p in outs)
    for p in sorted(figures, key=figures.get):
        print(f"coverage: {figures[p]:5.1f}%  {p}")
    low = below(figures, FLOOR)
    if low:
        print(f"coverage: below the {FLOOR:.0f}% floor: {', '.join(low)}")
        sys.exit(1)
    print(f"coverage: every core package at or above {FLOOR:.0f}% over {len(outs)} profiles")


main()
