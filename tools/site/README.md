# tools/site

Static site generator for the docs, manual and devlog.

```sh
pip install markdown
python3 tools/site/build.py          # writes site/
python3 tools/check/links.py site    # every link and anchor must resolve
```

- The source of truth is the markdown (plus example and proof files) in the repo. Never edit `site/` by hand.
- Project name, navigation, sources and man sections are set in the config block at the top of `build.py`.
- Pages with `man:` in their front matter get a man-page header and appear under their section in the manual sidebar.
- Code files under `examples/`, `proofs/` and `policy/` (`.yaml`, `.cedar`, `.go`, `.tla`, `.cfg`, `.als`) get their own highlighted pages.
- Mermaid diagrams render in the browser (loaded from the jsDelivr CDN). Offline, diagrams show as readable source.
- `style.css` and `app.js` are copied into `site/assets/`. `app.js` provides the theme toggle (light, dark, follows the system), syntax highlighting, search (press `/`), copy buttons and table-of-contents tracking.
- The site works from `file://`: the search index is a plain script, not a fetch.
- CI builds the site and runs the link check on every PR (SB-002, SB-009).
