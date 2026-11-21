package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/cell/wire/celltest"
)

func TestRefusesWithoutAConfiguration(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "cell.json")
	if err := os.WriteFile(bad, []byte(`{"tenant": "acme-logistics", "surprise": true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"--config", filepath.Join(t.TempDir(), "missing.json")}, {"--config", bad}, {"--nope"}} {
		var out bytes.Buffer
		if code := run(context.Background(), args, &out, nil); code != 1 || out.Len() == 0 {
			t.Errorf("run %v: exit %d, %q; want 1 with the reason", args, code, out.String())
		}
	}
}

// TestGracefulShutdown: a request in flight when the service is told to stop finishes,
// readiness goes first, and serve returns nil.
func TestGracefulShutdown(t *testing.T) {
	idp := oidctest.New("acme-logistics", "sb-acme-logistics", "oidc-secret")
	defer idp.Server.Close()
	cfg := celltest.Fixture(t, idp, "https://cell.example")
	cell, err := wire.Build(context.Background(), cfg, wire.Options{HTTP: idp.Server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cell, ln, ready) }()
	addr := <-ready
	get := func(path string) int {
		resp, err := http.Get("http://" + addr + path)
		if err != nil {
			return 0
		}
		defer func() { _ = resp.Body.Close() }()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}
	if code := get("/readyz"); code != http.StatusNoContent {
		t.Fatalf("readyz %d", code)
	}
	// A slow request: the body is sent only after the shutdown has begun.
	pr, pw := io.Pipe()
	slow := make(chan int, 1)
	go func() {
		resp, err := http.Post("http://"+addr+"/v1/ask", "application/json", pr)
		if err != nil {
			slow <- 0
			return
		}
		defer func() { _ = resp.Body.Close() }()
		slow <- resp.StatusCode
	}()
	time.Sleep(100 * time.Millisecond)
	stop()
	time.Sleep(100 * time.Millisecond)
	if cell.Ready() {
		t.Fatal("still ready after the stop signal")
	}
	_, _ = io.Copy(pw, strings.NewReader(`{"question":"x"}`))
	_ = pw.Close()
	if code := <-slow; code != http.StatusUnauthorized { // it was answered, not cut off
		t.Fatalf("in-flight request got %d, want its answer (401 without a session)", code)
	}
	if err := <-done; err != nil {
		t.Fatalf("serve: %v", err)
	}
	if code := get("/healthz"); code != 0 {
		t.Fatalf("still serving after shutdown: %d", code)
	}
}
