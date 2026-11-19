package auth

import (
	"io"
	"net/http"
	"net/url"
	"testing"

	"example.com/second-brain/internal/errs"
)

func errsEAUTH() *errs.Error { return errs.New(errs.EAUTH, "sign in again") }

func readAll(t *testing.T, r *http.Response) string {
	t.Helper()
	defer func() { _ = r.Body.Close() }()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
