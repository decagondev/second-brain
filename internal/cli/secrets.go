package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/errs"
)

// MaxScanFile is the largest file `sb secrets scan` reads; a larger one is a finding, not
// a skip, so nothing passes unread.
const MaxScanFile = 64 << 20

// secretsScan is the secret scanner (SB-411) as a command (sb-observability(7), Canaries
// and the secret scanner; SB-1808): `sb secrets scan [--known PATH]... PATH...`. It learns
// every value stored under each --known path (a vault directory, or a file such as
// db-password), then reads every regular file under each PATH and reports each line that
// holds a known value or a secret's shape (a private key, a URL with a password, a bearer
// token), by file, line and id, never the value. Any finding exits ELAW.
func (e *env) secretsScan(args []string) int {
	const cmd = "sb secrets scan"
	var known, paths []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--known":
			if i+1 >= len(args) {
				return e.fail(cmd, errs.New(errs.EUSAGE, "--known needs a path"))
			}
			known = append(known, args[i+1])
			i++
		case len(a) > 1 && a[0] == '-':
			return e.fail(cmd, errs.New(errs.EUSAGE, "unknown option %q", a))
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb secrets scan [--known PATH]... PATH..."))
	}
	var s canary.Scanner
	learnt := 0
	for _, k := range known {
		err := walkFiles(k, func(_ string, b []byte) {
			s.Learn(string(b))
			learnt++
		})
		if err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't read --known %s: %v", k, err))
		}
	}
	var findings []canary.Finding
	files := 0
	for _, p := range paths {
		err := walkFiles(p, func(name string, b []byte) {
			files++
			if b == nil {
				findings = append(findings, canary.Finding{Where: name, ID: "too_large_to_scan"})
				return
			}
			findings = append(findings, s.ScanAll(name, string(b))...)
		})
		if err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s: %v", p, err))
		}
	}
	sort.SliceStable(findings, func(a, b int) bool {
		if findings[a].Where != findings[b].Where {
			return findings[a].Where < findings[b].Where
		}
		return findings[a].Line < findings[b].Line
	})
	if e.g.JSON {
		e.json(map[string]any{"files": files, "known": learnt, "findings": findings})
	} else {
		for _, f := range findings {
			fmt.Fprintf(e.out, "%s:%d %s\n", f.Where, f.Line, f.ID)
		}
	}
	if len(findings) > 0 {
		return e.fail(cmd, errs.New(errs.ELAW, "%d secret(s) found in %d file(s) scanned; nothing here is safe to send", len(findings), files))
	}
	if !e.g.JSON && !e.g.Quiet {
		fmt.Fprintf(e.out, "secrets scan: %d file(s), %d stored value(s) known, no secret found\n", files, learnt)
	}
	return 0
}

// walkFiles calls fn for every regular file at or under root, in name order. Symbolic
// links are not followed. A file over MaxScanFile is passed with nil content.
func walkFiles(root string, fn func(name string, b []byte)) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > MaxScanFile {
			fn(p, nil)
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fn(p, b)
		return nil
	})
}

// selfhostCheckProxy checks an outbound proxy setting (sb-selfhost(7), Outbound proxy;
// SB-1806): `sb selfhost check-proxy URL` for SB_HTTPS_PROXY, which may not carry
// credentials, or `--file FILE` for SB_HTTPS_PROXY_FILE, which may. It prints the proxy's
// scheme and host only.
func (e *env) selfhostCheckProxy(args []string) int {
	const cmd = "sb selfhost check-proxy"
	var raw string
	var err error
	switch {
	case len(args) == 2 && args[0] == "--file":
		var b []byte
		if b, err = os.ReadFile(args[1]); err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", args[1]))
		}
		raw = string(b)
		err = box.CheckProxyFile(raw)
	case len(args) == 1 && args[0] != "" && args[0][0] != '-':
		raw = args[0]
		err = box.CheckProxy(raw)
	default:
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb selfhost check-proxy URL | --file FILE"))
	}
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"proxy": box.ProxyHost(raw)})
	}
	fmt.Fprintf(e.out, "proxy: %s\n", box.ProxyHost(raw))
	return 0
}

// selfhostCheckDSN checks the external database's connection string (sb-selfhost(7), A
// database on another machine; SB-1807): `sb selfhost check-dsn FILE [--service OUT]`. It
// prints the host only. --service writes the string as a pg_service.conf section named sb
// (mode 0600), which dbdump.sh and restore.sh give their Postgres client, so the password
// is never in a process's arguments or environment.
func (e *env) selfhostCheckDSN(args []string) int {
	const cmd = "sb selfhost check-dsn"
	var file, service string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--service":
			if i+1 >= len(args) {
				return e.fail(cmd, errs.New(errs.EUSAGE, "--service needs a file"))
			}
			service = args[i+1]
			i++
		case file == "" && a != "" && a[0] != '-':
			file = a
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb selfhost check-dsn FILE [--service OUT]"))
		}
	}
	if file == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, "usage: sb selfhost check-dsn FILE [--service OUT]"))
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", file))
	}
	params, err := box.DSNParams(string(b))
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	if service != "" {
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var sb strings.Builder
		sb.WriteString("[sb]\n")
		for _, k := range keys {
			sb.WriteString(k + "=" + params[k] + "\n")
		}
		if err := os.WriteFile(service, []byte(sb.String()), 0o600); err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't write %s", service))
		}
	}
	if e.g.JSON {
		return e.json(map[string]any{"host": params["host"]})
	}
	fmt.Fprintf(e.out, "database: %s\n", params["host"])
	return 0
}
