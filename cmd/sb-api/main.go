// Command sb-api is a cell's api service (sb-runtime(7), sb-api(3); SB-1502): it builds the
// cell from its configuration (sb-cell(5)) and serves every endpoint until SIGTERM.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/jobsvc"
	"example.com/second-brain/internal/cell/wire"
)

// Grace is how long a shutdown waits for requests in flight.
const Grace = 25 * time.Second

func main() {
	// An outbound proxy with credentials, from a file (SB-1806), before any HTTP client.
	if err := wire.ProxyFromFile(os.Getenv, os.Setenv); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "sb-api:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stderr, nil)
	stop()
	os.Exit(code)
}

// run starts the api and blocks until ctx ends. ready, when not nil, gets the address it
// listens on once it serves.
func run(ctx context.Context, args []string, stderr io.Writer, ready chan<- string) int {
	fs := flag.NewFlagSet("sb-api", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("config", "", "the cell's configuration file (sb-cell(5))")
	with := fs.String("with", "", "services to run in this process too: ingest, jobs, or ingest,jobs (sb-cell(5))")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	also, err := services(*with)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-api:", err)
		return 1
	}
	if *path == "" {
		_, _ = fmt.Fprintln(stderr, "sb-api: --config FILE is required")
		return 1
	}
	cfg, err := wire.Load(*path)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-api:", err)
		return 1
	}
	cell, err := wire.Build(ctx, cfg, wire.Options{})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-api:", err)
		return 1
	}
	ln, err := net.Listen("tcp", cfg.Listen.API)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-api:", err)
		return 1
	}
	bg, stopBG := context.WithCancel(ctx)
	var wg sync.WaitGroup
	ing := &ingestsvc.Service{Cell: cell}
	if also["ingest"] {
		wg.Add(1)
		go func() { defer wg.Done(); ing.Run(bg) }()
	}
	if also["jobs"] {
		wg.Add(1)
		go func() { defer wg.Done(); (&jobsvc.Service{Cell: cell, Ingest: ing}).Loop(bg) }()
	}
	wg.Add(1)
	go func() { defer wg.Done(); cell.Listen(bg) }() // SB-6005: the chat apps' websockets
	err = serve(ctx, cell, ln, ready)
	stopBG()
	wg.Wait()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-api:", err)
		return 1
	}
	return 0
}

// services parses --with.
func services(with string) (map[string]bool, error) {
	out := map[string]bool{}
	if with == "" {
		return out, nil
	}
	for _, s := range strings.Split(with, ",") {
		if s != "ingest" && s != "jobs" {
			return nil, fmt.Errorf("--with %q: name ingest, jobs or both", with)
		}
		out[s] = true
	}
	return out, nil
}

// serve serves the cell on ln until ctx ends, then stops taking requests (readiness goes
// first, so the platform stops routing) and lets those in flight finish.
func serve(ctx context.Context, cell *wire.Cell, ln net.Listener, ready chan<- string) error {
	srv := &http.Server{Handler: cell.Handler(), ReadHeaderTimeout: 10 * time.Second}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	if ready != nil {
		ready <- ln.Addr().String()
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
	}
	cell.Stop()
	sctx, cancel := context.WithTimeout(context.Background(), Grace)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return err
	}
	if err := <-done; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
