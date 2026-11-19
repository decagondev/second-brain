#!/usr/bin/env python3
"""SB-009: render every Mermaid diagram in the built site in headless Chromium; fail on any error.

Usage: python3 tools/check/diagrams.py site
Needs: pip install playwright && python3 -m playwright install chromium
       npm install --prefix tools/check        (pinned Mermaid, see package.json)
Set CHROME=/path/to/chrome to use an installed browser instead of Playwright's.
"""
import asyncio, os, sys
from pathlib import Path

from playwright.async_api import async_playwright

ROOT = Path(__file__).resolve().parents[2]
SITE = Path(sys.argv[1] if len(sys.argv) > 1 else ROOT / "site").resolve()
DIST = Path(os.environ.get("MERMAID_DIST", ROOT / "tools/check/node_modules/mermaid/dist"))

async def main():
    if not (DIST / "mermaid.esm.min.mjs").exists():
        sys.exit(f"diagrams.py: Mermaid not found at {DIST}; run npm install --prefix tools/check")
    pages = [p for p in sorted(SITE.rglob("*.html")) if 'class="mermaid"' in p.read_text()]
    total = bad = 0
    async with async_playwright() as pw:
        kw = {"executable_path": os.environ["CHROME"]} if os.environ.get("CHROME") else {}
        browser = await pw.chromium.launch(args=["--allow-file-access-from-files"], **kw)
        ctx = await browser.new_context()

        async def local(route):  # serve the pinned Mermaid instead of the CDN
            await route.fulfill(path=str(DIST / route.request.url.split("/dist/")[1]), content_type="text/javascript")
        await ctx.route("https://cdn.jsdelivr.net/npm/mermaid@11/dist/**", local)
        page = await ctx.new_page()
        for p in pages:
            await page.goto(p.as_uri())
            for _ in range(80):
                if await page.evaluate("[...document.querySelectorAll('pre.mermaid')].every(e => e.querySelector('svg'))"):
                    break
                await page.wait_for_timeout(100)
            res = await page.evaluate("""[...document.querySelectorAll('pre.mermaid')].map(e => {
                const s = e.querySelector('svg');
                return !s || s.getAttribute('aria-roledescription') === 'error' || e.textContent.includes('Syntax error');
            })""")
            for i, failed in enumerate(res):
                total += 1
                if failed:
                    bad += 1
                    print(f"diagrams: FAIL {p.relative_to(SITE)} diagram {i}")
        await browser.close()
    print(f"diagrams: {len(pages)} pages, {total} diagrams, {bad} failures")
    sys.exit(1 if bad or not total else 0)

asyncio.run(main())
