// Command sb-jobs is a cell's jobs service on its own (sb-runtime(7), Schedules; SB-1504):
// `sb-jobs --config FILE` runs every job on the local clock, and `sb-jobs --config FILE
// run JOB` runs one job and exits, as Cloud Scheduler asks for it. It shares the cell's
// stores with the api, so it runs only when they are durable (SB-1508); a cell whose
// stores are in memory runs jobs inside sb-api (`--with jobs`).
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
	"example.com/second-brain/internal/cell/jobsvc"
	"example.com/second-brain/internal/cell/wire"
)

func main() {
	// An outbound proxy with credentials, from a file (SB-1806), before any HTTP client.
	if err := wire.ProxyFromFile(os.Getenv, os.Setenv); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "sb-jobs:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("sb-jobs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("config", "", "the cell's configuration file (sb-cell(5))")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	rest := fs.Args()
	if len(rest) != 0 && (len(rest) != 2 || rest[0] != "run") {
		_, _ = fmt.Fprintln(stderr, "usage: sb-jobs --config FILE [run JOB]")
		return 1
	}
	cfg, err := wire.Load(*path)
	if err == nil {
		err = wire.Separate(cfg)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-jobs:", err)
		return 1
	}
	cell, err := wire.Build(ctx, cfg, wire.Options{})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-jobs:", err)
		return 1
	}
	svc := &jobsvc.Service{Cell: cell, Ingest: &ingestsvc.Service{Cell: cell}}
	if len(rest) == 2 {
		if err := svc.Run(ctx, rest[1]); err != nil {
			_, _ = fmt.Fprintln(stderr, "sb-jobs:", err)
			return 1
		}
		return 0
	}
	svc.Loop(ctx)
	return 0
}
