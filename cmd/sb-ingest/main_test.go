package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefusesInMemoryStores(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cell.json")
	cfg := `{"tenant": "acme-logistics", "manifest": "/m.yaml", "state": "/s", "vault": "/v", "policies": "/p",
 "listen": {"api": "127.0.0.1:8080"}, "base_url": "https://acme-logistics.sb.localhost",
 "oidc": {"client_id": "sb-acme-logistics"}, "model": {"provider": "extractive"}}`
	if err := os.WriteFile(p, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := run(context.Background(), []string{"--config", p}, &out); code != 1 || !strings.Contains(out.String(), "--with ingest") {
		t.Fatalf("exit %d: %s", code, out.String())
	}
}
