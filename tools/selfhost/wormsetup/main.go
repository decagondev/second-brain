// Command wormsetup makes the self-hosted cell's write-once bucket (sb-selfhost(7);
// SB-1603): it creates the bucket with object lock, sets seven years of compliance
// retention, checks the configuration, and proves the store enforces it by failing to
// delete a locked probe.
//
//	wormsetup --endpoint http://127.0.0.1:7070 --bucket sb-worm --creds-file FILE
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/backup/s3"
)

func main() {
	endpoint := flag.String("endpoint", "", "the store, http(s)://HOST:PORT")
	bucket := flag.String("bucket", "sb-worm", "the bucket")
	credsFile := flag.String("creds-file", "", `a file holding {"access_key","secret_key"}`)
	flag.Parse()
	creds, err := os.ReadFile(*credsFile)
	if err != nil || *endpoint == "" {
		fmt.Fprintln(os.Stderr, "wormsetup: --endpoint and a readable --creds-file are required")
		os.Exit(2)
	}
	b := s3.Bucket{Endpoint: *endpoint, BucketName: *bucket, HTTP: &http.Client{Timeout: 30 * time.Second},
		Creds: func(context.Context) (string, error) { return strings.TrimSpace(string(creds)), nil }}
	ctx := context.Background()
	if err := b.CreateLocked(ctx, 7); err != nil {
		fmt.Fprintln(os.Stderr, "wormsetup:", err)
		os.Exit(1)
	}
	if err := b.Enforced(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "wormsetup:", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, "wormsetup: bucket", *bucket, "is write-once for seven years, and the store enforces it")
}
