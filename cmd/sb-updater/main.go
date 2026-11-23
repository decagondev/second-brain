// Command sb-updater is a cell's updater job (sb-runtime(7), Updates are pulled; SB-1505):
// it reads the release the cell's channel should run, admits the bundle with the four
// checks of sb-release(7), attests each image and deploys each service by digest, then
// exits. A release that isn't admitted changes nothing and raises a Sev-1. It needs only
// the configuration's files, so it runs on its own.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/obs"
)

func main() {
	// An outbound proxy with credentials, from a file (SB-1806), before any HTTP client.
	if err := wire.ProxyFromFile(os.Getenv, os.Setenv); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "sb-updater:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sb-updater", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("config", "", "the cell's configuration file (sb-cell(5))")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	cfg, err := wire.Load(*path)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-updater:", err)
		return 1
	}
	job, err := wire.NewUpdater(cfg, wire.NewAlerter(cfg, stderr, obs.NewMetrics()), obs.NewMetrics())
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-updater:", err)
		return 1
	}
	res, err := job.Run(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-updater:", err)
		return 1
	}
	_ = json.NewEncoder(stdout).Encode(res)
	return 0
}
