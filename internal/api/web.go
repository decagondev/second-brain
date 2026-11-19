package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// CSP is the web UI's content security policy (sb-ask(3), The web UI): nothing loads
// from anywhere but the cell itself, and no one may frame it.
const CSP = "default-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'; object-src 'none'"

// Web serves the built web UI (web/dist from the release) at /. Paths that aren't files
// get index.html, so the app can route; /v1 is never served from here.
type Web struct {
	Files fs.FS
}

// Register mounts the UI on mux; the more specific /v1 patterns win.
func (h Web) Register(mux *http.ServeMux) {
	mux.Handle("GET /", h)
}

func (h Web) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hd := w.Header()
	hd.Set("Content-Security-Policy", CSP)
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Referrer-Policy", "no-referrer")
	if strings.HasPrefix(r.URL.Path, "/v1/") {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "index.html"
	}
	if st, err := fs.Stat(h.Files, name); err != nil || st.IsDir() {
		name = "index.html"
	}
	if name == "index.html" {
		hd.Set("Cache-Control", "no-cache")
	} else if strings.HasPrefix(name, "assets/") {
		hd.Set("Cache-Control", "public, max-age=31536000, immutable") // Vite names assets by hash
	}
	http.ServeFileFS(w, r, h.Files, name)
}
