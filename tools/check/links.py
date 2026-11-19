#!/usr/bin/env python3
"""Check every internal link and anchor in the built site. Exit 1 on any break.

Usage: python3 tools/check/links.py site
"""
import re, sys
from pathlib import Path

site = Path(sys.argv[1] if len(sys.argv) > 1 else "site")
bad, ids_cache = [], {}

def ids(p):
    if p not in ids_cache:
        ids_cache[p] = set(re.findall(r'id="([^"]+)"', p.read_text(encoding="utf-8")))
    return ids_cache[p]

pages = list(site.rglob("*.html"))
for f in pages:
    for href in re.findall(r'href="([^"]+)"', f.read_text(encoding="utf-8")):
        if re.match(r"^[a-z]+:", href):
            continue
        path, _, frag = href.partition("#")
        tgt = (f.parent / path).resolve() if path else f
        if not tgt.exists():
            bad.append((str(f.relative_to(site)), href))
        elif frag and tgt.suffix == ".html" and frag not in ids(tgt):
            bad.append((str(f.relative_to(site)), href + "  [missing anchor]"))
print(f"{len(pages)} pages, {len(bad)} broken links")
for b in sorted(set(bad))[:80]:
    print("  ", *b)
sys.exit(1 if bad else 0)
