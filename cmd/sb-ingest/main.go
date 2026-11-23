// Command sb-ingest is a cell's ingest service on its own (sb-ingest(7), sb-runtime(7);
// SB-1503). It shares the cell's stores with the api, so it runs only when they are
// durable (SB-1508); a cell whose stores are in memory runs ingest inside sb-api
// (`--with ingest`).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"example.com/second-brain/internal/cell/ingestsvc"
	"example.com/second-brain/internal/cell/wire"
)

func main() {
	// An outbound proxy with credentials, from a file (SB-1806), before any HTTP client.
	if err := wire.ProxyFromFile(os.Getenv, os.Setenv); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "sb-ingest:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("sb-ingest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("config", "", "the cell's configuration file (sb-cell(5))")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	cfg, err := wire.Load(*path)
	if err == nil {
		err = wire.Separate(cfg)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-ingest:", err)
		return 1
	}
	cell, err := wire.Build(ctx, cfg, wire.Options{})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-ingest:", err)
		return 1
	}
	(&ingestsvc.Service{Cell: cell}).Run(ctx)
	return 0
}
