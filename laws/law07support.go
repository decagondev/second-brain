package laws

import (
	"fmt"
	"math/rand"
	"strings"

	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/core/canary"
)

// LAW-07 for a self-hosted box (SB-1806, SB-1808): a support bundle never carries a stored
// secret, in whole or in part (a JSON object's secret field, one line of a key), and an
// outbound proxy's credentials never sit in a setting that isn't secret, nor in the error
// that refuses them. LAW-02 for an external database (SB-1807): the cell talks to a
// database on another machine only over TLS that checks the server's name.

// supportScan learns the stored values and returns the lines of text it finds them on.
type supportScan func(stored []string, text string) map[int]bool

func realSupportScan(stored []string, text string) map[int]bool {
	var s canary.Scanner
	for _, v := range stored {
		s.Learn(v)
	}
	out := map[int]bool{}
	for _, f := range s.ScanAll("bundle", text) {
		out[f.Line] = true
	}
	return out
}

func lawToken(r *rand.Rand, n int) string {
	const a = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = a[r.Intn(len(a))]
	}
	return string(b)
}

func law07Support(r *rand.Rand, scan supportScan) error {
	var stored, leaks []string
	for i := 0; i <= r.Intn(4); i++ {
		sec := lawToken(r, 16+r.Intn(24))
		switch r.Intn(3) {
		case 0: // a plain value: a token, a password
			stored = append(stored, sec)
			leaks = append(leaks, sec)
		case 1: // a JSON credential: its secret leaks alone
			stored = append(stored, fmt.Sprintf(`{"access_key":"sb-worm","secret_key":"%s"}`, sec))
			leaks = append(leaks, sec)
		default: // a multi-line key: one line leaks
			stored = append(stored, "-----BEGIN PRIVATE KEY-----\n"+sec+"\n"+lawToken(r, 20)+"\n-----END PRIVATE KEY-----")
			leaks = append(leaks, sec)
		}
	}
	clean := []string{"level=info msg=started service=api", "api running", "bucket=sb-worm", "image sb-api@sha256:" + lawHex([]byte{byte(r.Intn(256))}), "readyz 204"}
	var lines []string
	dirty := map[int]bool{}
	for i := 0; i < 10+r.Intn(20); i++ {
		if r.Intn(4) == 0 {
			l := leaks[r.Intn(len(leaks))]
			lines = append(lines, []string{"err=" + l, "level=error msg=\"upstream said " + l + "\"", l}[r.Intn(3)])
			dirty[len(lines)] = true
		} else {
			lines = append(lines, clean[r.Intn(len(clean))])
		}
	}
	got := scan(stored, strings.Join(lines, "\n"))
	for n := range dirty {
		if !got[n] {
			return fmt.Errorf("line %d carries a stored secret and the scan missed it: %q", n, lines[n-1])
		}
	}
	for n := range got {
		if !dirty[n] {
			return fmt.Errorf("line %d holds no secret and the scan flagged it: %q", n, lines[n-1])
		}
	}
	return nil
}

func law07Proxy(r *rand.Rand, check func(string) error) error {
	host := []string{"proxy.acme.example", "10.1.2.3", "egress"}[r.Intn(3)]
	base := fmt.Sprintf("%s://%%s%s:%d", []string{"http", "https"}[r.Intn(2)], host, 1024+r.Intn(60000))
	if err := check(fmt.Sprintf(base, "")); err != nil {
		return fmt.Errorf("a proxy with no credentials was refused: %v", err)
	}
	pw := lawToken(r, 8+r.Intn(16))
	err := check(fmt.Sprintf(base, "svc-"+lawToken(r, 4)+":"+pw+"@"))
	if err == nil {
		return fmt.Errorf("a proxy URL carrying credentials was accepted into a setting that isn't secret")
	}
	if strings.Contains(err.Error(), pw) {
		return fmt.Errorf("the refusal repeats the proxy's password: %v", err)
	}
	return nil
}

func law02DSN(r *rand.Rand, check func(string) error) error {
	host := []string{"db.acme.example", "10.20.0.7", "pg-1", "127.0.0.1", "localhost", "::1"}[r.Intn(6)]
	mode := []string{"", "disable", "allow", "prefer", "require", "verify-ca", "verify-full"}[r.Intn(7)]
	pw := lawToken(r, 12)
	var dsn string
	if r.Intn(2) == 0 {
		h := host
		if strings.Contains(h, ":") {
			h = "[" + h + "]"
		}
		dsn = fmt.Sprintf("postgres://sb:%s@%s:5432/sb", pw, h)
		if mode != "" {
			dsn += "?sslmode=" + mode
		}
	} else {
		dsn = fmt.Sprintf("host=%s port=5432 user=sb password=%s dbname=sb", host, pw)
		if mode != "" {
			dsn += " sslmode=" + mode
		}
	}
	local := host == "127.0.0.1" || host == "localhost" || host == "::1"
	err := check(dsn)
	switch {
	case mode == "verify-full" || local:
		if err != nil {
			return fmt.Errorf("%s with sslmode=%q was refused: %v", host, mode, err)
		}
	case err == nil:
		return fmt.Errorf("a database on another machine (%s) was accepted with sslmode=%q, which doesn't check its name", host, mode)
	case strings.Contains(err.Error(), pw):
		return fmt.Errorf("the refusal repeats the database's password: %v", err)
	}
	return nil
}

var realProxyCheck = box.CheckProxy
var realDSNCheck = box.CheckDSN
