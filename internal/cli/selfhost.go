package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/release/cosign"
	relfs "example.com/second-brain/internal/adapters/release/localfs"
	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/release"
	"example.com/second-brain/internal/errs"
)

// selfhostDoctor prints the verdict of every check over the facts the host step gathered
// (sb-selfhost(7), Doctor; SB-1801): `sb selfhost doctor [--facts FILE]`, stdin by default.
func (e *env) selfhostDoctor(args []string) int {
	path := "-"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--facts":
			if i+1 >= len(args) {
				return e.fail("sb selfhost doctor", errs.New(errs.EUSAGE, "--facts needs a file"))
			}
			path = args[i+1]
			i++
		default:
			return e.fail("sb selfhost doctor", errs.New(errs.EUSAGE, "unknown argument %q", args[i]))
		}
	}
	var b []byte
	var err error
	if path == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return e.fail("sb selfhost doctor", errs.New(errs.EINVAL, "can't read the facts: %v", err))
	}
	f, err := box.Parse(b)
	if err != nil {
		return e.fail("sb selfhost doctor", errs.New(errs.EINVAL, "%v", err))
	}
	lines := box.Check(f, time.Now())
	code := box.ExitCode(lines)
	if e.g.JSON {
		e.json(map[string]any{"verdict": box.Worst(lines), "checks": lines})
		return code
	}
	for _, l := range lines {
		fmt.Fprintf(e.out, "%-7s %-32s %s\n", l.Verdict, l.Check, l.Detail)
		if l.Fix != "" {
			fmt.Fprintf(e.out, "        fix: %s\n", l.Fix)
		}
	}
	fmt.Fprintf(e.out, "doctor: %s\n", box.Worst(lines))
	return code
}

// selfhostVerifyBundle checks an unpacked offline bundle before anything is loaded
// (sb-selfhost(7), Offline upgrades; SB-1804): the channel's release in DIR/releases is
// admitted exactly as the updater admits it, and each image the release names must be an
// archive in DIR/images that is exactly that image. It prints the archives to load, one
// per line, and nothing if any check fails.
func (e *env) selfhostVerifyBundle(args []string) int {
	const cmd = "sb selfhost verify-bundle"
	var dir, trusted, identity string
	channel := "stable"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--trusted", "--cosign-identity", "--channel":
			if i+1 >= len(args) {
				return e.fail(cmd, errs.New(errs.EUSAGE, "%s needs a value", a))
			}
			v := args[i+1]
			i++
			switch a {
			case "--trusted":
				trusted = v
			case "--cosign-identity":
				identity = v
			default:
				channel = v
			}
		default:
			if strings.HasPrefix(a, "-") || dir != "" {
				return e.fail(cmd, errs.New(errs.EUSAGE, "unknown argument %q", a))
			}
			dir = a
		}
	}
	if dir == "" || (trusted == "") == (identity == "") {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: %s DIR --trusted ed25519:KEY|--cosign-identity ID [--channel stable]", cmd))
	}
	var v release.Verifier = release.Ed25519Verifier{Trusted: trusted}
	if identity != "" {
		v = cosign.Verifier{Identity: identity}
	}
	ctx := context.Background()
	store := relfs.Dir{Root: filepath.Join(dir, "releases")}
	id, err := store.Channel(ctx, channel)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EUNSIGNED, "the bundle has no %s release: %v", channel, err))
	}
	m, _, err := release.Admit(ctx, store, id, v, nil)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EUNSIGNED, "%s isn't admitted: %v", id, err))
	}
	tars, _ := filepath.Glob(filepath.Join(dir, "images", "*.tar"))
	type load struct{ Image, ID, File string }
	var out []load
	for name, want := range m.Code {
		if !strings.HasPrefix(name, "image/") {
			continue
		}
		found := ""
		for _, t := range tars {
			f, err := os.Open(t)
			if err != nil {
				continue
			}
			err = release.VerifyImageTar(f, want)
			_ = f.Close()
			if err == nil {
				found = t
				break
			}
		}
		if found == "" {
			return e.fail(cmd, errs.New(errs.EUNSIGNED, "no archive in the bundle is exactly %s (sha256:%s)", name, want))
		}
		out = append(out, load{Image: name, ID: "sha256:" + want, File: found})
	}
	if len(out) == 0 {
		return e.fail(cmd, errs.New(errs.EUNSIGNED, "%s names no images", id))
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Image < out[b].Image })
	if e.g.JSON {
		return e.json(map[string]any{"release": id, "load": out})
	}
	for _, l := range out {
		fmt.Fprintln(e.out, l.File)
	}
	return 0
}
