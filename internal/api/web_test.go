package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWeb(t *testing.T) {
	files := fstest.MapFS{
		"index.html":         {Data: []byte("<!doctype html><title>Second Brain</title>")},
		"assets/app-abc.js":  {Data: []byte("console.log(1)")},
		"assets/app-abc.css": {Data: []byte("body{}")},
	}
	mux := http.NewServeMux()
	Ask{Agent: &fakeAgent{}, Release: "rel_x"}.Register(mux)
	Web{Files: files}.Register(mux)
	get := func(p string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		return w
	}
	for _, p := range []string{"/", "/some/route", "/etc/passwd", "/assets"} {
		w := get(p)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Second Brain") {
			t.Errorf("%s: %d %q", p, w.Code, w.Body)
		}
		if w.Header().Get("Content-Security-Policy") != CSP || w.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: headers %v", p, w.Header())
		}
	}
	if w := get("/../../etc/passwd"); w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != "/etc/passwd" {
		t.Errorf("dot-dot path: %d %v (the mux must clean it first)", w.Code, w.Header())
	}
	w := get("/assets/app-abc.js")
	if w.Code != 200 || w.Body.String() != "console.log(1)" || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset: %d %v", w.Code, w.Header())
	}
	if w := get("/v1/nope"); w.Code != 404 {
		t.Errorf("/v1 fell through to the UI: %d", w.Code)
	}
	if w := get("/v1/traces/tr_1"); w.Code != 401 {
		t.Errorf("/v1/traces without a scope: %d, want 401 from the API", w.Code)
	}
	if !strings.Contains(CSP, "default-src 'self'") || !strings.Contains(CSP, "frame-ancestors 'none'") {
		t.Error("CSP must match sb-ask(3)")
	}
}

// SB-2301: the page carries the manifest's language, which the UI starts in.
func TestWebLanguage(t *testing.T) {
	files := fstest.MapFS{"index.html": {Data: []byte(`<!doctype html><html lang="en"><title>Second Brain</title></html>`)}}
	lang := "de"
	mux := http.NewServeMux()
	Web{Files: files, Language: func() string { return lang }}.Register(mux)
	for _, c := range []struct{ lang, want string }{{"de", `lang="de"`}, {"fr", `lang="fr"`}, {"", `lang="en"`}, {"en", `lang="en"`}} {
		lang = c.lang
		for _, p := range []string{"/", "/q/sh_x"} {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), c.want) || w.Header().Get("Content-Security-Policy") != CSP || w.Header().Get("Cache-Control") != "no-cache" {
				t.Errorf("%q %s: %d %q %v", c.lang, p, w.Code, w.Body, w.Header())
			}
		}
	}
}
