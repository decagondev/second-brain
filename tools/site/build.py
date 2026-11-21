#!/usr/bin/env python3
"""Build the Second Brain docs, manual and devlog site from the repository's markdown.

Usage:  python3 tools/site/build.py [--out site]
Needs:  pip install markdown

Single source of truth: every page is generated from files in the repo
(README, docs/, manual/, living/, epics/, examples/, proofs/, policy/).
Never edit site/ by hand. Same shape as the Keel site generator, so both
projects' docs read the same way.
"""
import argparse, html, json, re, shutil, sys
from pathlib import Path

try:
    import markdown
except ImportError:
    sys.exit("error: python package 'markdown' is required (pip install markdown)")

ROOT = Path(__file__).resolve().parents[2]
HERE = Path(__file__).resolve().parent

# ---------------------------------------------------------------- project config
PROJECT = "Second Brain"
BRAND = "second brain"
MARK = "◈"
TAGLINE = "if the platform says it, it holds"
SOURCES = ["README.md", "CONTRIBUTING.md", "docs", "manual", "living", "epics", "examples", "proofs", "policy", "laws", "tests", "evals"]
SKIP = set()
CODE_EXT = {".yaml": "yaml", ".yml": "yaml", ".cedar": "cedar", ".cedarschema": "cedar", ".go": "go",
            ".tla": "tla", ".cfg": "tla", ".als": "alloy", ".py": "python", ".expected": "text", ".json": "json"}
NAV = [  # (label, href relative to site root, section key)
    ("Home", "index.html", "home"),
    ("Tour", "manual/sb-tour.7.html", "tour"),
    ("Manual", "manual/README.html", "manual"),
    ("Design", "docs/README.html", "design"),
    ("Plan", "epics/README.html", "epics"),
    ("Devlog", "devlog.html", "living"),
    ("Examples", "examples/README.html", "examples"),
]
MAN_SECTIONS = [("7", "Concepts"), ("1", "Commands"), ("3", "API"), ("5", "File formats")]

# ---------------------------------------------------------------- parsing

def front_matter(text):
    meta = {}
    if text.startswith("---\n"):
        end = text.find("\n---\n", 4)
        if end != -1:
            for line in text[4:end].splitlines():
                if ":" in line:
                    k, v = line.split(":", 1)
                    meta[k.strip()] = v.strip().strip('"')
            text = text[end + 5:]
    return meta, text

def section_of(rel):
    return {"docs": "design", "manual": "manual", "living": "living", "epics": "epics",
            "examples": "examples", "proofs": "design", "policy": "design", "laws": "design", "tests": "design", "evals": "design"}.get(rel.parts[0], "home")

def out_name(rel):
    s = str(rel)
    return s[:-3] + ".html" if s.endswith(".md") else s + ".html"

def rewrite_links(body):
    def fix(m):
        url = m.group(2)
        if re.match(r"^[a-z]+:", url) or url.startswith("#"):
            return m.group(0)
        path, _, frag = url.partition("#")
        if path.endswith(".md"):
            path = path[:-3] + ".html"
        elif Path(path).suffix in CODE_EXT:
            path = path + ".html"
        elif path.endswith("/") and path:
            path = path + "README.html"
        return f'{m.group(1)}="{path}{"#" + frag if frag else ""}"'
    return re.sub(r'(href)="([^"]+)"', fix, body)

def plain_title(t):
    return t.replace("`", "")

def render_md(text):
    text = re.sub(r"^```(\w+) run$", r"```\1", text, flags=re.M)  # `run` tag is for CI, not the reader
    mermaids = []
    def grab(m):
        mermaids.append(m.group(1))
        return f"\n\nSBMERMAID{len(mermaids) - 1}\n\n"
    text = re.sub(r"```mermaid\n(.*?)```", grab, text, flags=re.S)
    md = markdown.Markdown(extensions=["fenced_code", "tables", "toc", "sane_lists", "attr_list"],
                           extension_configs={"toc": {"toc_depth": "2-3", "permalink": False}})
    body = md.convert(text)
    for i, src in enumerate(mermaids):
        block = f'<div class="mermaid-wrap"><pre class="mermaid">{html.escape(src)}</pre></div>'
        body = body.replace(f"<p>SBMERMAID{i}</p>", block)
    body = body.replace("<li>[ ] ", '<li class="task"><span class="box"></span>').replace(
        "<li>[x] ", '<li class="task done"><span class="box">✓</span>')
    return rewrite_links(body), md.toc_tokens

def toc_html(tokens):
    items = []
    def walk(ts):
        for t in ts:
            if t["level"] in (2, 3):
                items.append(f'<li class="l{t["level"]}"><a href="#{t["id"]}">{t["name"]}</a></li>')
            walk(t.get("children", []))
    walk(tokens)
    return "<ul>" + "".join(items) + "</ul>" if len(items) > 1 else ""

def plain(htmltext, n=6000):
    t = re.sub(r'<pre class="mermaid">.*?</pre>', " ", htmltext, flags=re.S)  # diagrams aren't searchable prose
    t = re.sub(r"<[^>]+>", " ", t)
    return re.sub(r"\s+", " ", html.unescape(t)).strip()[:n]

# ---------------------------------------------------------------- collection

def collect():
    pages = []
    for src in SOURCES:
        p = ROOT / src
        if not p.exists():
            continue
        files = [p] if p.is_file() else sorted(f for f in p.rglob("*") if f.is_file() and (f.suffix == ".md" or f.suffix in CODE_EXT) and "__pycache__" not in f.parts and f.name != "__init__.py")
        for f in files:
            rel = f.relative_to(ROOT)
            if str(rel) in SKIP:
                continue
            raw = f.read_text(encoding="utf-8")
            if f.suffix in CODE_EXT:
                lang = CODE_EXT[f.suffix]
                meta = {"title": rel.name, "summary": f"Source file ({lang})"}
                up, back = f.parent, "README.md"
                while not (up / "README.md").exists() and up != ROOT:  # nearest README that exists
                    up, back = up.parent, "../" + back
                body_md = f"# {rel.name}\n\n```{lang}\n{raw.rstrip()}\n```\n\n[Back]({back})\n"
            else:
                meta, body_md = front_matter(raw)
                if "title" not in meta:
                    h = re.search(r"^# (.+)$", body_md, re.M)
                    meta["title"] = h.group(1) if h else rel.stem
            pages.append({"rel": rel, "out": out_name(rel), "meta": meta, "md": body_md, "section": section_of(rel)})
    return pages

# ---------------------------------------------------------------- sidebar

def sidebar(page, pages, prefix):
    sec = page["section"]
    def link(p, label=None):
        cur = ' class="current"' if p["out"] == page["out"] else ""
        return f'<a{cur} href="{prefix}{p["out"]}">{html.escape(plain_title(label or p["meta"]["title"]))}</a>'
    def under(top, sub=None):
        return [p for p in pages if p["rel"].parts[0] == top and (sub is None or (len(p["rel"].parts) > 2 and p["rel"].parts[1] == sub))]
    groups = []
    if sec == "manual":
        man = [p for p in pages if p["section"] == "manual"]
        groups.append(("Manual", [p for p in man if p["rel"].name == "README.md"]))
        order = (ROOT / "manual/README.md").read_text()
        rank = {m: i for i, m in enumerate(re.findall(r"\]\(([^)]+\.md)\)", order))}
        for s, label in MAN_SECTIONS:
            ps = sorted([p for p in man if p["meta"].get("man") == s], key=lambda p: rank.get(p["rel"].name, 999))
            groups.append((f"({s}) {label}", ps))
    elif sec == "design":
        d = under("docs")
        groups.append(("Design", sorted([p for p in d if len(p["rel"].parts) == 2], key=lambda p: (p["rel"].name != "README.md", str(p["rel"])))))
        groups.append(("Architecture", sorted(under("docs", "architecture"), key=lambda p: str(p["rel"]))))
        groups.append(("Decisions", sorted(under("docs", "adr"), key=lambda p: (p["rel"].name != "README.md", str(p["rel"])))))
        groups.append(("Proofs and policy", sorted(under("proofs") + under("policy"), key=lambda p: (p["rel"].name != "README.md", str(p["rel"])))))
        groups.append(("Laws, tests and evals", sorted(under("laws") + under("tests") + under("evals"), key=lambda p: str(p["rel"]))))
    elif sec == "epics":
        e = [p for p in pages if p["section"] == "epics"]
        groups.append(("Plan", [p for p in e if len(p["rel"].parts) == 2] + [x for x in pages if str(x["rel"]) == "docs/03-prd.md"]))
        groups.append(("Epics", sorted([p for p in e if p["rel"].name == "README.md" and len(p["rel"].parts) == 3], key=lambda p: str(p["rel"]))))
        if len(page["rel"].parts) == 3:
            ep = page["rel"].parts[1]
            groups.append(("Tickets in " + ep.split("-")[0], sorted([p for p in e if len(p["rel"].parts) == 3 and p["rel"].parts[1] == ep and p["rel"].name != "README.md"], key=lambda p: p["rel"].name)))
    elif sec == "living":
        groups.append(("Living documents", [{"out": "devlog.html", "meta": {"title": "Devlog timeline"}}] + sorted([p for p in pages if p["section"] == "living"], key=lambda p: p["meta"]["title"])))
    elif sec == "examples":
        groups.append(("Examples", sorted([p for p in pages if p["section"] == "examples"], key=lambda p: (p["rel"].name != "README.md", str(p["rel"])))))
    else:
        start = ("README.md", "CONTRIBUTING.md", "docs/00-problem-statement.md", "manual/sb-tour.7.md", "manual/sb-onboarding.7.md", "docs/03-prd.md")
        groups.append(("Start here", sorted([p for p in pages if str(p["rel"]) in start], key=lambda p: start.index(str(p["rel"])))))
    out = []
    for title, ps in groups:
        if ps:
            out.append(f'<div class="group"><h4>{html.escape(title)}</h4>' + "".join(link(p) for p in ps) + "</div>")
    return "".join(out)

# ---------------------------------------------------------------- templates

def shell(title, prefix, active, body, side="", toc="", extra_class=""):
    nav = "".join(f'<a class="{"active" if key == active else ""}" href="{prefix}{href}">{label}</a>' for label, href, key in NAV)
    return f"""<!doctype html>
<html lang="en" data-theme="light">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(plain_title(title))} · {PROJECT}</title>
<link rel="stylesheet" href="{prefix}assets/style.css">
<script>try{{const t=localStorage.getItem('sb-theme');if(t)document.documentElement.dataset.theme=t;else if(matchMedia('(prefers-color-scheme: dark)').matches)document.documentElement.dataset.theme='dark'}}catch(e){{}}</script>
</head>
<body class="{extra_class}">
<a class="skip" href="#main">Skip to content</a>
<header class="top">
  <button class="menu" aria-label="Menu" onclick="document.body.classList.toggle('nav-open')">☰</button>
  <a class="brand" href="{prefix}index.html"><span class="mark">{MARK}</span> {BRAND}</a>
  <nav class="tabs" aria-label="Main">{nav}</nav>
  <div class="search"><input id="q" type="search" placeholder="Search docs  /" autocomplete="off" aria-label="Search"><div id="results" role="listbox"></div></div>
  <button class="theme" aria-label="Toggle light and dark theme" onclick="sbToggleTheme()">◐</button>
</header>
<div class="layout{' wide' if not side else ''}">
  {f'<aside class="side" aria-label="Section">{side}</aside>' if side else ''}
  <main class="content" id="main">{body}</main>
  {f'<aside class="toc" aria-label="On this page"><h4>On this page</h4>{toc}</aside>' if toc else ''}
</div>
<footer>{PROJECT} · generated from the repository's markdown by <code>tools/site/build.py</code> · {TAGLINE}</footer>
<script>window.SB_PREFIX="{prefix}";</script>
<script src="{prefix}assets/search-index.js"></script>
<script src="{prefix}assets/app.js"></script>
<script type="module">
import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
window.sbMermaid = mermaid; window.sbRenderMermaid && window.sbRenderMermaid();
</script>
</body></html>"""

def man_header(meta):
    t = meta["title"].upper()
    return f'<div class="manhead"><span>{html.escape(t)}</span><span>{PROJECT} Manual</span><span>{html.escape(t)}</span></div>'

def page_html(page, pages, prefix):
    body, toks = render_md(page["md"])
    meta = page["meta"]
    head = man_header(meta) if page["section"] == "manual" and meta.get("man") else ""
    parent = str(page["rel"].parent)
    crumbs = f'<div class="crumbs"><a href="{prefix}index.html">{BRAND}</a> / {html.escape(parent) if parent != "." else ""}</div>'
    body = head + crumbs + body + f'<p class="srcline">Source: <code>{page["rel"]}</code></p>'
    active = page["section"]
    if str(page["rel"]) == "manual/sb-tour.7.md":
        active = "tour"
    if str(page["rel"]) == "docs/03-prd.md":
        active = "epics"
    return shell(meta["title"], prefix, active, body, sidebar(page, pages, prefix), toc_html(toks)), body

# ---------------------------------------------------------------- special pages

def devlog_page(pages):
    p = next(p for p in pages if str(p["rel"]) == "living/DEVLOG.md")
    parts = re.split(r"^## ", p["md"], flags=re.M)
    cards = []
    for e in parts[1:]:
        title, _, rest = e.partition("\n")
        m = re.match(r"(\d{4}-\d{2}-\d{2})\s*[:—-]?\s*(.*)", title)
        date, name = (m.group(1), m.group(2)) if m else ("", title)
        body, _ = render_md(rest)
        body = re.sub(r'href="(?![a-z]+:|#)([^"]+)"', r'href="living/\1"', body)  # entries are written relative to living/
        cards.append((date, f'<article class="entry"><div class="date">{date}</div><div class="card"><h2>{html.escape(name)}</h2>{body}</div></article>'))
    cards = [c for _, c in sorted(cards, key=lambda dc: dc[0], reverse=True)]  # newest first even if an entry was filed out of order; stable within a day
    hero = (f'<div class="crumbs"><a href="index.html">{BRAND}</a> / devlog</div><h1>Devlog</h1>'
            '<p class="lede">The running story of Second Brain, newest first. Every PR adds an entry, and so does every significant design step. '
            'The <b>Learned</b> line is the most valuable part.</p>'
            '<p class="views"><a href="living/DEVLOG.html">Plain view</a> · <a href="living/BUGS_AND_MITIGATIONS.html">Bugs and mitigations</a> · '
            '<a href="living/ASSURANCE_LEDGER.html">Assurance ledger</a> · <a href="living/RISKS.html">Risks</a> · <a href="living/OPEN_QUESTIONS.html">Open questions</a></p>')
    side = sidebar({"section": "living", "out": "devlog.html", "rel": Path("devlog")}, pages, "")
    return shell("Devlog", "", "living", hero + '<div class="timeline">' + "".join(cards) + "</div>", side), cards

def index_page(pages, latest):
    sample, _ = render_md("""```yaml
tenant: acme-logistics
channel: stable
idp: oidc://login.acme.example
region: eu
connectors: { docs: sharepoint, chat: slack }
agents: [ask, draft-reply]
```

```sh
sb tenant check  acme.yaml
sb tenant apply  acme.yaml
sb tenant verify acme.yaml
```""")
    prd = (ROOT / "docs/03-prd.md").read_text()
    waves = re.findall(r"^\| (W\d+) \| ([^|]+) \| ([^|]+) \| ([^|]+) \|$", prd, re.M)
    wave_html = "".join(f'<a class="wave" href="docs/03-prd.html#{w.lower()}-{re.sub(r"[^a-z0-9]+", "-", n.strip().lower()).strip("-")}"><b>{w}</b><span>{html.escape(n.strip())}</span><small>{html.escape(l.strip())}</small></a>' for w, n, l, _ in waves)
    laws = re.findall(r"^### (LAW-\d\d): (.+?) \{#", (ROOT / "manual/sb-laws.7.md").read_text(), re.M)
    law_html = "".join(f'<a class="law" href="manual/sb-laws.7.html#{i.lower()}"><b>{i}</b> {html.escape(t)}</a>' for i, t in laws)
    n_tickets = sum(1 for p in pages if p["section"] == "epics" and p["rel"].name.startswith("SB-"))
    n_epics = sum(1 for p in pages if p["section"] == "epics" and len(p["rel"].parts) == 3 and p["rel"].name == "README.md")
    n_man = sum(1 for p in pages if p["meta"].get("man"))
    cards = [
        ("Take the tour", "manual/sb-tour.7.html", "Fifteen minutes from a six-line manifest to a cited answer to an approved action."),
        ("Read the manual", "manual/README.html", f"{n_man} man pages: commands, API, file formats and concepts. The manual is the spec."),
        ("Understand the design", "docs/README.html", "Problem, principles, an architecture document for every subsystem, ADRs and the threat model."),
        ("See the plan", "epics/README.html", f"{n_epics} epics and {n_tickets} tickets with subtasks, in {len(waves)} waves. Formal models land in Wave 0."),
        ("Onboard a company", "manual/sb-onboarding.7.html", "The day 0 to 90 runbook a company's own IT team follows, without calling anyone."),
        ("Follow the devlog", "devlog.html", "Decisions, lessons and progress, newest first."),
    ]
    card_html = "".join(f'<a class="tile" href="{h}"><h3>{t}</h3><p>{d}</p></a>' for t, h, d in cards)
    body = f"""
<section class="hero">
  <div class="pitch">
    <p class="kicker">A company second brain for every acquisition</p>
    <h1>Cited answers.<br>Gated actions.<br>Permissions that hold.</h1>
    <p class="lede">{PROJECT} gives every company the parent organisation acquires an isolated, AI-first knowledge platform, configured by <b>one file</b> and stood up by its own IT team with <b>three commands</b>. Answers cite exact document versions the asker can open <b>right now</b>. Agents draft; humans send or approve.</p>
    <div class="cta"><a class="btn primary" href="manual/sb-tour.7.html">Start the tour →</a><a class="btn" href="docs/00-problem-statement.html">Why Second Brain?</a></div>
  </div>
  <div class="sample">{sample}<div class="verdict"><span class="ok">✓</span> LAW-01 · LAW-02 · LAW-03 · LAW-14 pass on real data · 0 unknowns</div></div>
</section>
<section class="facts">
  <div><b>Unknown means no</b><span>A timeout, error or missing source is a refusal (<code>EPERM</code>), never a pass.</span></div>
  <div><b>The source decides</b><span>Every chunk is checked live with the source before the model sees it.</span></div>
  <div><b>Ran means receipt</b><span>An action is done only when the target system says so.</span></div>
  <div><b>One cell per company</b><span>Own database, index, keys and region. One template, one release train.</span></div>
</section>
<h2 class="band">Where to go</h2>
<section class="tiles">{card_html}</section>
<h2 class="band">Roadmap</h2>
<section class="waves">{wave_html}</section>
<h2 class="band">The laws</h2>
<section class="laws">{law_html}</section>
<h2 class="band">Latest from the devlog</h2>
<section class="latest">{latest}<p><a href="devlog.html">All entries →</a></p></section>
"""
    return shell(PROJECT, "", "home", body, extra_class="home")

# ---------------------------------------------------------------- main

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=str(ROOT / "site"))
    out = Path(ap.parse_args().out)
    if out.exists():
        shutil.rmtree(out)
    (out / "assets").mkdir(parents=True)
    for a in ("style.css", "app.js"):
        shutil.copy(HERE / a, out / "assets" / a)
    pages = collect()
    index = []
    for p in pages:
        prefix = "../" * (len(Path(p["out"]).parts) - 1)
        doc, body = page_html(p, pages, prefix)
        dest = out / p["out"]
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(doc, encoding="utf-8")
        index.append({"t": plain_title(p["meta"]["title"]), "u": p["out"], "s": p["meta"].get("summary", ""), "k": p["section"], "x": plain(body)})
    dl, cards = devlog_page(pages)
    (out / "devlog.html").write_text(dl, encoding="utf-8")
    latest = cards[0] if cards else ""
    (out / "index.html").write_text(index_page(pages, latest), encoding="utf-8")
    index.append({"t": "Devlog timeline", "u": "devlog.html", "s": "Newest-first development story", "k": "living", "x": ""})
    (out / "assets" / "search-index.js").write_text("window.SB_INDEX=" + json.dumps(index, separators=(",", ":")) + ";", encoding="utf-8")
    print(f"built {len(pages) + 2} pages into {out}")

if __name__ == "__main__":
    main()
