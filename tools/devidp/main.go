// Command devidp is a development OIDC identity provider for a local cell (`make cell`,
// SB-1508). It signs in whichever directory user it is told to, with no password, so it
// listens on loopback only and must never front a real cell:
//
//	go run ./tools/devidp --listen 127.0.0.1:8081 --tenant acme-logistics --client sb-acme-logistics --secret-file FILE
//
// `POST /dev/user?id=usr_…` changes who signs in next.
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/oidc/oidctest"
)

var userRe = regexp.MustCompile(`^usr_[A-Za-z0-9_]{1,64}$`)

func main() {
	listen := flag.String("listen", "127.0.0.1:8081", "loopback address to listen on")
	tenant := flag.String("tenant", "", "the tenant whose users sign in")
	client := flag.String("client", "", "the cell's OIDC client id")
	secretFile := flag.String("secret-file", "", "a file holding the client secret (the cell reads the same from its vault)")
	user := flag.String("user", "usr_dev", "who signs in until /dev/user changes it")
	flag.Parse()
	host, _, err := net.SplitHostPort(*listen)
	if err != nil || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		fmt.Fprintln(os.Stderr, "devidp: --listen must be a loopback address; this IdP has no passwords")
		os.Exit(2)
	}
	secret, err := os.ReadFile(*secretFile)
	if err != nil || *tenant == "" || *client == "" {
		fmt.Fprintln(os.Stderr, "devidp: --tenant, --client and a readable --secret-file are required")
		os.Exit(2)
	}
	idp := oidctest.NewAt(*tenant, *client, strings.TrimSpace(string(secret)), "http://"+*listen)
	idp.User = *user
	mux := http.NewServeMux()
	mux.Handle("/", idp.Handler())
	mux.HandleFunc("POST /dev/user", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if !userRe.MatchString(id) {
			http.Error(w, "id must be usr_…", http.StatusBadRequest)
			return
		}
		idp.User = id
		_, _ = fmt.Fprintln(w, "next sign-in:", id)
	})
	srv := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	fmt.Fprintln(os.Stderr, "devidp: issuer", idp.Issuer())
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "devidp:", err)
		os.Exit(1)
	}
}
