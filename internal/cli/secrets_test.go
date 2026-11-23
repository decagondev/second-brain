package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSecretsScan(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	write(t, filepath.Join(vault, "api/vault/tenants/acme-logistics/scim"), "Vt4kQm9ZpX2wLr7NcB8s")
	write(t, filepath.Join(vault, "jobs/vault/tenants/acme-logistics/worm"), `{"access_key":"sb-worm","secret_key":"Hq3nW8xPz1KmT5vR"}`)
	out := filepath.Join(dir, "bundle")
	write(t, filepath.Join(out, "logs/api.log"), "level=info msg=started\nlevel=info bucket=sb-worm\n")
	write(t, filepath.Join(out, "ps.txt"), "api running\n")

	run := func(args ...string) (int, string, string) {
		var o, e bytes.Buffer
		code := Run(args, &o, &e)
		return code, o.String(), e.String()
	}
	if code, o, e := run("secrets", "scan", "--known", vault, out); code != 0 || !strings.Contains(o, "2 file(s), 2 stored value(s) known, no secret found") {
		t.Fatalf("clean: %d %q %q", code, o, e)
	}
	write(t, filepath.Join(out, "logs/jobs.log"), "ok\nlevel=error msg=\"s3: Hq3nW8xPz1KmT5vR rejected\"\n")
	write(t, filepath.Join(out, "doctor.txt"), "proxy http://olga:pw12345@proxy:3128\n")
	code, o, e := run("secrets", "scan", "--known", vault, out)
	if code != 1 || !strings.Contains(o, "logs/jobs.log:2 sec_") || !strings.Contains(o, "doctor.txt:1 shape_url_password") || !strings.Contains(e, "[ELAW]") {
		t.Fatalf("leak: %d %q %q", code, o, e)
	}
	if strings.Contains(o+e, "Hq3nW8xPz1KmT5vR") || strings.Contains(o+e, "pw12345") {
		t.Fatalf("the scan printed a secret: %q %q", o, e)
	}
	if code, _, _ := run("secrets", "scan"); code != 2 {
		t.Fatalf("no path: %d", code)
	}
	if code, _, _ := run("secrets", "scan", "--known", filepath.Join(dir, "nope"), out); code != 2 {
		t.Fatalf("missing known: %d", code)
	}
}

func TestSelfhostCheckProxyAndDSN(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) (int, string, string) {
		var o, e bytes.Buffer
		code := Run(args, &o, &e)
		return code, o.String(), e.String()
	}
	if code, o, _ := run("selfhost", "check-proxy", "http://proxy.acme.example:3128"); code != 0 || o != "proxy: http://proxy.acme.example:3128\n" {
		t.Fatalf("plain: %d %q", code, o)
	}
	if code, o, e := run("selfhost", "check-proxy", "http://u:s3cretpw@proxy:3128"); code != 2 || strings.Contains(o+e, "s3cretpw") || !strings.Contains(e, "SB_HTTPS_PROXY_FILE") {
		t.Fatalf("creds: %d %q %q", code, o, e)
	}
	pf := filepath.Join(dir, "proxy")
	write(t, pf, "http://u:s3cretpw@proxy:3128\n")
	if code, o, _ := run("selfhost", "check-proxy", "--file", pf); code != 0 || o != "proxy: http://proxy:3128\n" {
		t.Fatalf("file: %d %q", code, o)
	}
	df := filepath.Join(dir, "dsn")
	write(t, df, "postgres://sb:dbs3cret@db.acme.example:5432/sb?sslmode=require\n")
	if code, o, e := run("selfhost", "check-dsn", df); code != 2 || strings.Contains(o+e, "dbs3cret") || !strings.Contains(e, "verify-full") {
		t.Fatalf("dsn: %d %q %q", code, o, e)
	}
	write(t, df, "postgres://sb:dbs3cret@db.acme.example:5432/sb?sslmode=verify-full\n")
	if code, o, _ := run("selfhost", "check-dsn", df); code != 0 || o != "database: db.acme.example\n" {
		t.Fatalf("dsn ok: %d %q", code, o)
	}
}

func TestSelfhostCheckDSNService(t *testing.T) {
	dir := t.TempDir()
	df, sf := filepath.Join(dir, "dsn"), filepath.Join(dir, "svc.conf")
	write(t, df, "postgres://sb:dbs3cret@db.acme.example:5432/sb?sslmode=verify-full\n")
	var o, e bytes.Buffer
	if code := Run([]string{"selfhost", "check-dsn", df, "--service", sf}, &o, &e); code != 0 {
		t.Fatalf("%d %q", code, e.String())
	}
	b, err := os.ReadFile(sf)
	if err != nil || string(b) != "[sb]\ndbname=sb\nhost=db.acme.example\npassword=dbs3cret\nport=5432\nsslmode=verify-full\nuser=sb\n" {
		t.Fatalf("service file: %q %v", b, err)
	}
	if st, _ := os.Stat(sf); st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode())
	}
}
