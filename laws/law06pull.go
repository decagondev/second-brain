package laws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"example.com/second-brain/internal/adapters/release/httpstore"
)

// LAW-06 from a tenant's own account (SB-707): the same admission law, with the cell reading
// the release store over HTTPS with only its read-only pull token, as a tenant-custody cell
// does. The factory has no way into the cell; the store refuses any other token.

// NewPullReleaseWorld builds the system under test; call the returned func to stop the store.
var NewPullReleaseWorld = func() (ReleaseWorld, func()) {
	w := newReleaseWorld(false)
	const token = "pull-gamma-parts"
	srv := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer "+token {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rel, name, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/releases/"), "/")
		if !ok {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		b, err := w.store.Get(r.Context(), rel, name)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = rw.Write(b)
	}))
	w.pull = httpstore.Store{Base: srv.URL, HTTP: srv.Client(), Token: func(context.Context) (string, error) { return token, nil }}
	return w, srv.Close
}
