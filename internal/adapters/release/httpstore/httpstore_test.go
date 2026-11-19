package httpstore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenAndPaths(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer srv.Close()
	s := Store{Base: srv.URL, HTTP: srv.Client(), Token: func(context.Context) (string, error) { return "good", nil }}
	b, err := s.Get(context.Background(), "rel_2027-03-02_abc", "artefacts/sb")
	if err != nil || string(b) != "/releases/rel_2027-03-02_abc/artefacts/sb" {
		t.Fatalf("%q %v", b, err)
	}
	for _, bad := range [][2]string{{"rel_x", "../secret"}, {"..", "bundle.json"}, {"rel_x", "a b"}, {"rel_x", "/abs"}} {
		if _, err := s.Get(context.Background(), bad[0], bad[1]); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	s.Token = func(context.Context) (string, error) { return "factory", nil }
	if _, err := s.Get(context.Background(), "rel_x", "bundle.json"); err == nil {
		t.Fatal("a refused token read the store")
	}
}
