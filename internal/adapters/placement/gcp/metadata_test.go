package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetadataToken(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" || r.URL.Path != "/computeMetadata/v1/instance/service-accounts/default/token" {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		calls++
		_, _ = w.Write([]byte(`{"access_token":"ya29.test","expires_in":3600}`))
	}))
	defer srv.Close()
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	m := &MetadataToken{Base: srv.URL, HTTP: srv.Client(), Now: func() time.Time { return now }}
	for i := 0; i < 3; i++ {
		if tok, err := m.Token(context.Background()); err != nil || tok != "ya29.test" {
			t.Fatalf("%q %v", tok, err)
		}
	}
	if calls != 1 {
		t.Fatalf("%d calls, want the token reused", calls)
	}
	now = now.Add(59*time.Minute + time.Second)
	if _, err := m.Token(context.Background()); err != nil || calls != 2 {
		t.Fatalf("a token near expiry wasn't refreshed: %d %v", calls, err)
	}
	bad := &MetadataToken{Base: srv.URL + "/nope", HTTP: srv.Client()}
	if _, err := bad.Token(context.Background()); err == nil {
		t.Fatal("an error answer gave a token")
	}
}
