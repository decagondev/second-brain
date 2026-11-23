package box

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckProxy(t *testing.T) {
	for _, ok := range []string{"http://proxy.acme.example:3128", "https://proxy:8443/", " http://10.0.0.5:3128\n"} {
		if err := CheckProxy(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "proxy:3128", "socks5://proxy:1080", "http://", "http://proxy:3128/x", "http://a b", "http://proxy:3128?x=1"} {
		if err := CheckProxy(bad); !errors.Is(err, ErrSetting) {
			t.Errorf("%q: want ErrSetting, got %v", bad, err)
		}
	}
	err := CheckProxy("http://olga:hunter22secret@proxy:3128")
	if !errors.Is(err, ErrSetting) || strings.Contains(err.Error(), "hunter22secret") || !strings.Contains(err.Error(), "SB_HTTPS_PROXY_FILE") {
		t.Fatalf("credentials: %v", err)
	}
	if err := CheckProxyFile("http://olga:hunter22secret@proxy:3128"); err != nil {
		t.Fatalf("a file may carry credentials: %v", err)
	}
	if got := ProxyHost("http://olga:hunter22secret@proxy:3128"); got != "http://proxy:3128" {
		t.Fatalf("ProxyHost = %q", got)
	}
}

func TestCheckDSN(t *testing.T) {
	for _, ok := range []string{
		"postgres://sb:pw@db.acme.example:5432/sb?sslmode=verify-full",
		"postgresql://sb:pw@db.acme.example/sb?sslmode=verify-full&sslrootcert=/etc/sb/ca/db.crt",
		"host=db.acme.example port=5432 user=sb password=pw dbname=sb sslmode=verify-full",
		"postgres://sb:pw@127.0.0.1:5432/sb?sslmode=disable",
		"host=localhost user=sb dbname=sb",
		"postgres://sb:pw@[::1]:5432/sb",
	} {
		if err := CheckDSN(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{
		"",
		"postgres://sb:topsecretpw@db.acme.example/sb",
		"postgres://sb:topsecretpw@db.acme.example/sb?sslmode=require",
		"postgres://sb:topsecretpw@db.acme.example/sb?sslmode=verify-ca",
		"host=db.acme.example password=topsecretpw sslmode=disable",
		"host=/var/run/postgresql dbname=sb",
		"mysql://sb:topsecretpw@db/sb",
		"dbname=sb password=topsecretpw",
		"host=db.acme.example garbage",
	} {
		err := CheckDSN(bad)
		if !errors.Is(err, ErrSetting) {
			t.Errorf("%q: want ErrSetting, got %v", bad, err)
			continue
		}
		if strings.Contains(err.Error(), "topsecretpw") {
			t.Errorf("%q: the error repeats the password: %v", bad, err)
		}
	}
	if got := DSNHost("postgres://sb:pw@db.acme.example:5432/sb?sslmode=verify-full"); got != "db.acme.example" {
		t.Fatalf("DSNHost = %q", got)
	}
}

func TestDSNParams(t *testing.T) {
	got, err := DSNParams("postgres://sb:p%40ss@db.acme.example:6432/cell?sslmode=verify-full&sslrootcert=/etc/sb/ca/db.crt")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"user": "sb", "password": "p@ss", "host": "db.acme.example", "port": "6432", "dbname": "cell", "sslmode": "verify-full", "sslrootcert": "/etc/sb/ca/db.crt"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s = %q, want %q", k, got[k], v)
		}
	}
	kv, err := DSNParams("host=db.acme.example user=sb password='pw' sslmode=verify-full")
	if err != nil || kv["password"] != "pw" || kv["host"] != "db.acme.example" {
		t.Fatalf("kv: %v %v", kv, err)
	}
	if _, err := DSNParams("postgres://sb:pw@db.acme.example/sb?sslmode=require"); !errors.Is(err, ErrSetting) {
		t.Fatalf("unverified: %v", err)
	}
	if _, err := DSNParams("postgres://sb:pw@db.acme.example/sb?sslmode=verify-full&Bad-Key=x"); !errors.Is(err, ErrSetting) {
		t.Fatalf("bad key: %v", err)
	}
}
