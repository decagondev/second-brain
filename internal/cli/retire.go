package cli

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/errs"
)

// selfhostRetireStatement writes a retired box's signed statement (sb-selfhost(7),
// Retiring a box; SB-1907) from the cell's audit export, taken just before the box was
// destroyed. It prints the statement's key: the private half is made here and dropped.
func (e *env) selfhostRetireStatement(args []string) int {
	const cmd = "sb selfhost retire-statement"
	const usage = "usage: sb selfhost retire-statement --tenant T --export FILE (--anchor-seed FILE --check | --cell-key ed25519:KEY --host H --destroyed TEXT... [--kept TEXT]... --out FILE)"
	check := false
	var rest []string
	for _, a := range e.localArgs(args) {
		if a == "--check" {
			check = true
		} else {
			rest = append(rest, a)
		}
	}
	opts, pos, err := flags(cmd, rest, "--tenant", "--host", "--export", "--anchor-seed", "--cell-key", "--destroyed", "--kept", "--out")
	if err != nil {
		return e.fail(cmd, err)
	}
	tenant, out := one(opts, "--tenant"), one(opts, "--out")
	if len(pos) > 0 || tenant == "" || one(opts, "--export") == "" || (one(opts, "--anchor-seed") == "") == (one(opts, "--cell-key") == "") {
		return e.fail(cmd, errs.New(errs.EUSAGE, usage))
	}
	if !check && (out == "" || len(opts["--destroyed"]) == 0) {
		return e.fail(cmd, errs.New(errs.EUSAGE, usage))
	}
	exp, err := os.ReadFile(one(opts, "--export"))
	if err != nil {
		return e.fail(cmd, errs.New(errs.EUSAGE, "cannot read the export: %v", err))
	}
	cellKey := one(opts, "--cell-key")
	if p := one(opts, "--anchor-seed"); p != "" {
		if cellKey, err = keyOfSeed(tenant, p); err != nil {
			return e.fail(cmd, err)
		}
	}
	last, n, err := box.LastAnchor(exp, tenant, cellKey)
	if err != nil {
		return e.fail(cmd, errs.New(errs.ECHAIN, "%v", err))
	}
	if check {
		// Before anything is destroyed: the export verifies under the cell's own key.
		if e.g.JSON {
			return e.json(map[string]any{"cell_key": cellKey, "last_seq": last.Seq, "last_hash": last.Hash, "rows": n})
		}
		fmt.Fprintln(e.out, cellKey)
		return 0
	}
	r, err := box.Retirement{Tenant: tenant, Host: one(opts, "--host"), Destroyed: opts["--destroyed"], Kept: opts["--kept"],
		LastAnchor: last, AuditRows: n}.Sign(rand.Reader, time.Now())
	if err != nil {
		return e.fail(cmd, errs.New(errs.EUSAGE, "%v", err))
	}
	if err := writeAtomic(out, func(w io.Writer) error { _, err := w.Write(r.JSON()); return err }); err != nil {
		return e.fail(cmd, errs.New(errs.EUSAGE, "cannot write %s: %v", out, err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"statement": out, "key": r.Key, "last_seq": last.Seq, "last_hash": last.Hash})
	}
	fmt.Fprintf(e.out, "statement: %s\nkey: %s\nlast anchor: seq %d, hash %s\n", out, r.Key, last.Seq, last.Hash)
	return 0
}

// keyOfSeed is the cell's anchor key from its seed file (32 bytes, base64).
func keyOfSeed(tenant, path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errs.New(errs.EUSAGE, "cannot read the anchor seed: %v", err)
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return "", errs.New(errs.EUSAGE, "the anchor seed isn't base64")
	}
	s, err := audit.NewSigner(tenant, seed)
	if err != nil {
		return "", errs.New(errs.EUSAGE, "%v", err)
	}
	return s.Key(), nil
}

// selfhostVerifyRetirement checks a statement under the key printed when it was made.
func (e *env) selfhostVerifyRetirement(args []string) int {
	const cmd = "sb selfhost verify-retirement"
	opts, pos, err := flags(cmd, args, "--key")
	if err != nil {
		return e.fail(cmd, err)
	}
	if len(pos) != 1 || one(opts, "--key") == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb selfhost verify-retirement FILE --key ed25519:KEY"))
	}
	b, err := os.ReadFile(pos[0])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EUSAGE, "cannot read %s", pos[0]))
	}
	r, err := box.VerifyRetirement(b, one(opts, "--key"))
	if err != nil {
		return e.fail(cmd, errs.New(errs.ECHAIN, "%v", err))
	}
	if e.g.JSON {
		return e.json(r)
	}
	fmt.Fprintf(e.out, "verify-retirement: %s retired %s at %s; last audit row %d, hash %s; signature good\n", r.Host, r.Tenant, r.At, r.LastAnchor.Seq, r.LastAnchor.Hash)
	for _, d := range r.Destroyed {
		fmt.Fprintf(e.out, "  destroyed: %s\n", d)
	}
	for _, k := range r.Kept {
		fmt.Fprintf(e.out, "  kept: %s\n", k)
	}
	return 0
}
