package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// MetadataBase is the metadata server a Cloud Run service or job asks for its own token.
const MetadataBase = "http://metadata.google.internal"

// MetadataToken is the running service account's access token, from the metadata server
// (SB-1505): the updater job deploys with its own identity, never a key file. A token is
// reused until a minute before it expires.
type MetadataToken struct {
	Base string // "" is MetadataBase
	HTTP *http.Client
	Now  func() time.Time

	mu  sync.Mutex
	tok string
	exp time.Time
}

// Token returns a valid access token.
func (m *MetadataToken) Token(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now
	if m.Now != nil {
		now = m.Now
	}
	if m.tok != "" && now().Before(m.exp.Add(-time.Minute)) {
		return m.tok, nil
	}
	base := m.Base
	if base == "" {
		base = MetadataBase
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/computeMetadata/v1/instance/service-accounts/default/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	hc := m.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("gcp: metadata token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gcp: metadata token: HTTP %d", resp.StatusCode)
	}
	var v struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&v); err != nil || v.AccessToken == "" {
		return "", fmt.Errorf("gcp: metadata token: no access_token")
	}
	m.tok, m.exp = v.AccessToken, now().Add(time.Duration(v.ExpiresIn)*time.Second)
	return m.tok, nil
}
