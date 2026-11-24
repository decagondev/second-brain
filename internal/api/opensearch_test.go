package api

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// SB-3806: a valid OpenSearch 1.1 description pointing at the Documents tab, the words as
// q; none without a base URL.
func TestOpenSearch(t *testing.T) {
	files := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Second Brain</title>")}}
	mux := http.NewServeMux()
	Web{Files: files, Base: "https://brain.acme.example/"}.Register(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/opensearch.xml", nil))
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/opensearchdescription+xml") {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
	var d struct {
		XMLName   xml.Name `xml:"http://a9.com/-/spec/opensearch/1.1/ OpenSearchDescription"`
		ShortName string   `xml:"ShortName"`
		URL       struct {
			Type     string `xml:"type,attr"`
			Template string `xml:"template,attr"`
		} `xml:"Url"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatalf("%v: %s", err, w.Body)
	}
	if d.ShortName != "Second Brain" || d.URL.Type != "text/html" || d.URL.Template != "https://brain.acme.example/?tab=documents&q={searchTerms}" {
		t.Fatalf("%+v", d)
	}
	if len(d.ShortName) > 16 {
		t.Fatal("ShortName is at most 16 characters")
	}
	mux = http.NewServeMux()
	Web{Files: files}.Register(mux)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/opensearch.xml", nil))
	if strings.Contains(w.Body.String(), "OpenSearchDescription") {
		t.Fatal("served without a base URL")
	}
}
