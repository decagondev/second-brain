// Second Brain docs site behaviour: theme, syntax highlighting (yaml, cedar, go, tla, alloy, json, http, sh),
// copy buttons, search, TOC tracking, mermaid. Same behaviour as the Keel docs site.
(function () {
  "use strict";
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => Array.from(r.querySelectorAll(s));
  const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

  // ---- theme
  window.sbToggleTheme = function () {
    const next = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    try { localStorage.setItem("sb-theme", next); } catch (e) {}
    rerenderMermaid();
  };

  // ---- highlighters: one small tokenizer, keyword sets per language
  const LANGS = {
    yaml:  { com: /#[^\n]*/, kw: "true false null" },
    cedar: { com: /\/\/[^\n]*/, kw: "when unless principal action resource context in has is like if then else entity appliesTo namespace type", spec: "permit forbid" },
    go:    { com: /\/\/[^\n]*|\/\*[\s\S]*?\*\//, kw: "package import return if else for range switch case default struct interface map chan go defer var const nil true false make append len error string int bool byte", spec: "func type" },
    tla:   { com: /\\\*[^\n]*|\(\*[\s\S]*?\*\)/, kw: "MODULE EXTENDS CONSTANTS CONSTANT VARIABLES VARIABLE ASSUME LET IN IF THEN ELSE CASE OTHER EXCEPT UNCHANGED ENABLED SUBSET UNION DOMAIN CHOOSE TRUE FALSE BOOLEAN Nat SPECIFICATION INVARIANTS INVARIANT PROPERTIES PROPERTY INIT NEXT CHECK_DEADLOCK", spec: "Init Next Spec TypeOK" },
    alloy: { com: /--[^\n]*|\/\/[^\n]*|\/\*[\s\S]*?\*\//, kw: "module sig abstract one lone some no all set pred fun extends in and or not implies let disj for", spec: "fact assert check run" },
    json:  { com: null, kw: "true false null" },
    http:  { com: null, kw: "GET POST PUT PATCH DELETE" },
    sh:    { com: /(?:^|[ \t])#[^\n]*/, kw: "sb make python3 pip curl jq git go java open xdg-open export" },
    python: { com: /#[^\n]*/, kw: "def return if elif else for in while import from as class lambda not and or None True False yield with", spec: "def class" },
    text:  { com: null, kw: "" },
  };
  LANGS.toml = LANGS.yaml;
  const ID_RE = "\\b(?:LAW|SB|ADR|US|BUG|Q|R)-\\d+(?:\\.\\d+)?\\b|\\bE[A-Z]{3,}\\b";
  const STR_RE = '"(?:\\\\.|[^"\\\\\\n])*"';
  function highlight(src, lang) {
    const L = LANGS[lang]; if (!L) return esc(src);
    const kw = new Set((L.kw || "").split(" ").filter(Boolean)), spec = new Set((L.spec || "").split(" ").filter(Boolean));
    const keyed = lang === "yaml" || lang === "json" || lang === "toml";
    const re = new RegExp([
      L.com ? "(" + L.com.source + ")" : "(\\u0000)",       // 1 comments
      "(" + STR_RE + ")",                                   // 2 strings
      "(" + ID_RE + ")",                                    // 3 ids, error codes
      "(\\b\\d[\\d_.]*\\b)",                                // 4 numbers
      keyed ? "([A-Za-z_][\\w-]*(?=\\s*:(?:\\s|$)))" : "(\\u0000)", // 5 keys
      "([A-Za-z_]\\w*)",                                    // 6 words
    ].join("|"), "gm");
    let out = "", last = 0, m;
    while ((m = re.exec(src))) {
      if (!m[0]) { re.lastIndex++; continue; }
      out += esc(src.slice(last, m.index));
      const t = m[0];
      let cls = null;
      if (m[1]) cls = "com"; else if (m[2]) cls = "str"; else if (m[3]) cls = "attr"; else if (m[4]) cls = "num";
      else if (m[5]) cls = "type"; else if (spec.has(t)) cls = "spec"; else if (kw.has(t)) cls = "kw";
      out += cls ? `<span class="tk-${cls}">${esc(t)}</span>` : esc(t);
      last = m.index + t.length;
    }
    return out + esc(src.slice(last));
  }
  $$("pre > code").forEach((c) => {
    const lang = (c.className.match(/language-(\w+)/) || [])[1];
    if (lang && LANGS[lang]) c.innerHTML = highlight(c.textContent, lang);
    const b = document.createElement("button");
    b.className = "copy"; b.textContent = "copy";
    b.onclick = () => { navigator.clipboard && navigator.clipboard.writeText(c.textContent); b.textContent = "copied"; setTimeout(() => (b.textContent = "copy"), 1200); };
    c.parentElement.appendChild(b);
  });

  // ---- search
  const q = $("#q"), res = $("#results"), P = window.SB_PREFIX || "";
  let sel = -1;
  function search(term) {
    term = term.trim().toLowerCase();
    if (term.length < 2) { res.classList.remove("open"); return; }
    const words = term.split(/\s+/);
    const scored = [];
    for (const p of window.SB_INDEX || []) {
      const t = p.t.toLowerCase(), s = p.s.toLowerCase(), x = p.x.toLowerCase();
      let score = 0, ok = true;
      for (const w of words) {
        const st = t.includes(w) ? 10 : 0, ss = s.includes(w) ? 4 : 0, sx = x.includes(w) ? 1 : 0;
        if (!st && !ss && !sx) { ok = false; break; }
        score += st + ss + sx;
      }
      if (ok) scored.push([score + (t.startsWith(words[0]) ? 5 : 0), p]);
    }
    scored.sort((a, b) => b[0] - a[0]);
    res.innerHTML = scored.slice(0, 12).map(([, p]) => {
      const i = p.x.toLowerCase().indexOf(words[0]);
      const snip = i >= 0 ? "…" + p.x.slice(Math.max(0, i - 40), i + 80) + "…" : p.s;
      return `<a href="${P}${p.u}"><span class="k">${esc(p.k)}</span>${esc(p.t)}<small>${esc(snip)}</small></a>`;
    }).join("") || `<a><small>No results for “${esc(term)}”</small></a>`;
    sel = -1; res.classList.add("open");
  }
  if (q) {
    q.addEventListener("input", () => search(q.value));
    q.addEventListener("keydown", (e) => {
      const items = $$("a", res);
      if (e.key === "ArrowDown") { sel = Math.min(sel + 1, items.length - 1); e.preventDefault(); }
      else if (e.key === "ArrowUp") { sel = Math.max(sel - 1, 0); e.preventDefault(); }
      else if (e.key === "Enter" && items[Math.max(sel, 0)]) { location.href = items[Math.max(sel, 0)].href; }
      else if (e.key === "Escape") { res.classList.remove("open"); q.blur(); }
      items.forEach((a, i) => a.classList.toggle("sel", i === sel));
    });
    document.addEventListener("keydown", (e) => { if (e.key === "/" && document.activeElement !== q) { e.preventDefault(); q.focus(); } });
    document.addEventListener("click", (e) => { if (!e.target.closest(".search")) res.classList.remove("open"); });
  }

  // ---- TOC highlighting
  const tocLinks = $$(".toc a");
  if (tocLinks.length && "IntersectionObserver" in window) {
    const map = new Map(tocLinks.map((a) => [decodeURIComponent(a.getAttribute("href").slice(1)), a]));
    const obs = new IntersectionObserver((ents) => {
      ents.forEach((en) => { if (en.isIntersecting) { tocLinks.forEach((a) => a.classList.remove("on")); const a = map.get(en.target.id); a && a.classList.add("on"); } });
    }, { rootMargin: "-60px 0px -70% 0px" });
    map.forEach((_, id) => { const h = document.getElementById(id); h && obs.observe(h); });
  }

  // ---- mermaid (module script sets window.sbMermaid; offline: diagrams stay as readable source)
  const sources = $$("pre.mermaid").map((el) => el.textContent);
  function themeVars() {
    const dark = document.documentElement.dataset.theme === "dark";
    return { theme: "base", themeVariables: dark
      ? { background: "#111918", primaryColor: "#16302d", primaryTextColor: "#dfe7e4", primaryBorderColor: "#3fd0bd", lineColor: "#7b8885", secondaryColor: "#1c2440", tertiaryColor: "#141c1b", fontFamily: "ui-sans-serif, sans-serif" }
      : { background: "#fdfdfa", primaryColor: "#e6f3f1", primaryTextColor: "#17201f", primaryBorderColor: "#0f766e", lineColor: "#6f7a77", secondaryColor: "#e9edf8", tertiaryColor: "#f6f6f2", fontFamily: "ui-sans-serif, sans-serif" } };
  }
  async function render() {
    const m = window.sbMermaid; if (!m) return;
    m.initialize(Object.assign({ startOnLoad: false, securityLevel: "strict", flowchart: { htmlLabels: true } }, themeVars()));
    const els = $$("pre.mermaid");
    els.forEach((el, i) => { el.removeAttribute("data-processed"); el.textContent = sources[i]; });
    try { await m.run({ nodes: els }); } catch (e) { console.warn("mermaid:", e); }
  }
  function rerenderMermaid() { render(); }
  window.sbRenderMermaid = render;
  if (window.sbMermaid) render();
})();
