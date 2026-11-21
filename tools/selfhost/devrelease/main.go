// Command devrelease publishes a demo release of this machine's images into a local
// release store (sb-selfhost(7); SB-1604): it signs a bundle made by tools/release/bundle
// with a key that belongs to this machine alone, writes a gate record, and points the
// channel at it. A cell trusts that key only when its configuration names it, which only
// the demo configuration does; the factory's releases are signed in CI (sb-release(7)).
//
//	devrelease --bundle bundle.json --store DIR --key-file FILE --gate-results FILE [--channel stable]
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strings"

	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/core/release"
)

func key(path string) (release.Ed25519, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		seed := make([]byte, ed25519.SeedSize)
		if _, err := rand.Read(seed); err != nil {
			return release.Ed25519{}, err
		}
		if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(seed)), 0o600); err != nil {
			return release.Ed25519{}, err
		}
		return release.Ed25519{Priv: ed25519.NewKeyFromSeed(seed)}, nil
	}
	if err != nil {
		return release.Ed25519{}, err
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return release.Ed25519{}, fmt.Errorf("%s isn't a 32-byte seed in base64", path)
	}
	return release.Ed25519{Priv: ed25519.NewKeyFromSeed(seed)}, nil
}

func main() {
	bundle := flag.String("bundle", "", "bundle.json from tools/release/bundle")
	store := flag.String("store", "", "the release store directory")
	keyFile := flag.String("key-file", "", "this machine's demo release key (made if missing)")
	results := flag.String("gate-results", "", "the gate's results file, whose SHA-256 the gate record carries")
	channel := flag.String("channel", "stable", "the channel to point at the release")
	flag.Parse()
	if err := run(*bundle, *store, *keyFile, *results, *channel); err != nil {
		fmt.Fprintln(os.Stderr, "devrelease:", err)
		os.Exit(1)
	}
}

func run(bundlePath, store, keyFile, resultsPath, channel string) error {
	if bundlePath == "" || store == "" || keyFile == "" || resultsPath == "" {
		return errors.New("--bundle, --store, --key-file and --gate-results are required")
	}
	k, err := key(keyFile)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		return err
	}
	var m release.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if err := m.Check(); err != nil {
		return err
	}
	results, err := os.ReadFile(resultsPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	d, err := m.Digest()
	if err != nil {
		return err
	}
	sig, _ := k.Sign(ctx, d)
	g := release.GateRecord{Release: m.ID, Passed: true, ResultsSHA256: release.SHA256(results)}
	gd, err := g.Digest()
	if err != nil {
		return err
	}
	gsig, _ := k.Sign(ctx, gd)
	st := relfs.Dir{Root: store}
	for name, v := range map[string]any{"bundle.sig": sig, "gate.json": g, "gate.sig": gsig} {
		b, _ := json.Marshal(v)
		if err := st.Put(m.ID, name, b); err != nil {
			return err
		}
	}
	if err := st.Put(m.ID, "bundle.json", raw); err != nil {
		return err
	}
	if err := st.SetChannel(channel, m.ID); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", m.ID, k.Public())
	return nil
}
