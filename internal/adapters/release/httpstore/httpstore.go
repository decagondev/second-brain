// Package httpstore is the release store a cell pulls from over HTTPS (sb-release(7), Cells
// in a tenant's account; SB-707): GET BASE/releases/ID/NAME with a read-only pull token
// from the cell's own vault. The factory never connects into the cell.
package httpstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"example.com/second-brain/internal/core/release"
)

// Store reads one release store.
type Store struct {
	Base  string
	HTTP  *http.Client
	Token func(ctx context.Context) (string, error) // vault://tenants/T/release-pull
}

var (
	_    release.Store = Store{}
	safe               = regexp.MustCompile(`^[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*$`)
)

// MaxObject is the largest object a cell will read.
const MaxObject = 256 << 20

// Get implements release.Store.
func (s Store) Get(ctx context.Context, rel, name string) ([]byte, error) {
	if !safe.MatchString(rel) || !safe.MatchString(name) || strings.Contains(rel+"/"+name, "..") {
		return nil, fmt.Errorf("httpstore: bad object %q/%q", rel, name)
	}
	tok, err := s.Token(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Base+"/releases/"+url.PathEscape(rel)+"/"+name, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("httpstore: HTTP %d for %s/%s", resp.StatusCode, rel, name)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxObject+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxObject {
		return nil, fmt.Errorf("httpstore: %s/%s is larger than %d bytes", rel, name, MaxObject)
	}
	return b, nil
}
