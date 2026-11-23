package wire

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProxyFromFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "https-proxy")
	if err := os.WriteFile(f, []byte("http://svc-sb:p4ssw0rd-proxy@proxy.acme.example:3128\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(env map[string]string) (map[string]string, error) {
		set := map[string]string{}
		err := ProxyFromFile(func(k string) string { return env[k] }, func(k, v string) error { set[k] = v; return nil })
		return set, err
	}
	if set, err := run(map[string]string{}); err != nil || len(set) != 0 {
		t.Fatalf("no file: %v %v", set, err)
	}
	set, err := run(map[string]string{"SB_HTTPS_PROXY_FILE": f})
	if err != nil || set["HTTPS_PROXY"] != "http://svc-sb:p4ssw0rd-proxy@proxy.acme.example:3128" {
		t.Fatalf("file: %v %v", set, err)
	}
	if _, err := run(map[string]string{"SB_HTTPS_PROXY_FILE": f, "HTTPS_PROXY": "http://other:3128"}); !errors.Is(err, ErrConfig) {
		t.Fatalf("both: %v", err)
	}
	if _, err := run(map[string]string{"SB_HTTPS_PROXY_FILE": filepath.Join(dir, "missing")}); !errors.Is(err, ErrConfig) {
		t.Fatalf("missing: %v", err)
	}
	bad := filepath.Join(dir, "bad")
	_ = os.WriteFile(bad, []byte("socks5://svc-sb:p4ssw0rd-proxy@proxy:1080"), 0o600)
	if _, err := run(map[string]string{"SB_HTTPS_PROXY_FILE": bad}); !errors.Is(err, ErrConfig) || strings.Contains(err.Error(), "p4ssw0rd") {
		t.Fatalf("bad: %v", err)
	}
}
