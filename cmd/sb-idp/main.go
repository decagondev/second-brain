// Command sb-idp is the bundled OIDC identity provider for a self-hosted cell with no IdP
// of its own (sb-idp(1); SB-1605).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"example.com/second-brain/internal/adapters/oidc/bundled"
)

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

const usage = `usage: sb-idp serve --issuer URL --data DIR --tenant T --client ID --secret-file FILE --redirect URL [--listen ADDR] [--behind-proxy]
       sb-idp user add|disable --data DIR --email EMAIL --scim URL --scim-token-file FILE
       sb-idp user list --data DIR`

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch {
	case len(args) >= 1 && args[0] == "serve":
		return serve(args[1:], stderr)
	case len(args) >= 2 && args[0] == "user":
		return user(args[1], args[2:], stdin, stdout, stderr)
	}
	_, _ = fmt.Fprintln(stderr, usage)
	return 2
}

func serve(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("sb-idp serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	issuer := fs.String("issuer", "", "")
	data := fs.String("data", "", "")
	tenant := fs.String("tenant", "", "")
	client := fs.String("client", "", "")
	secretFile := fs.String("secret-file", "", "")
	redirect := fs.String("redirect", "", "")
	listen := fs.String("listen", "0.0.0.0:8082", "")
	behind := fs.Bool("behind-proxy", false, "")
	if fs.Parse(args) != nil || *issuer == "" || *data == "" || *tenant == "" || *client == "" || *secretFile == "" || *redirect == "" {
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
	if !strings.HasPrefix(*issuer, "https://") || !strings.HasPrefix(*redirect, "https://") {
		_, _ = fmt.Fprintln(stderr, "sb-idp: the issuer and the redirect must be https://; the proxy serves both")
		return 2
	}
	secret, err := os.ReadFile(*secretFile)
	if err != nil || strings.TrimSpace(string(secret)) == "" {
		_, _ = fmt.Fprintln(stderr, "sb-idp: the client secret can't be read")
		return 1
	}
	st := &bundled.Store{Dir: *data}
	key, err := st.Key()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-idp:", err)
		return 1
	}
	s := bundled.New(*issuer, *tenant, *client, strings.TrimSpace(string(secret)), *redirect, st, key)
	s.BehindProxy = *behind
	srv := &http.Server{Addr: *listen, Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	_, _ = fmt.Fprintln(stderr, "sb-idp: issuer", *issuer)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_, _ = fmt.Fprintln(stderr, "sb-idp:", err)
		return 1
	}
	return 0
}

func user(cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sb-idp user", flag.ContinueOnError)
	fs.SetOutput(stderr)
	data := fs.String("data", "", "")
	email := fs.String("email", "", "")
	scimURL := fs.String("scim", "", "")
	tokenFile := fs.String("scim-token-file", "", "")
	if fs.Parse(args) != nil || *data == "" {
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
	st := &bundled.Store{Dir: *data}
	if cmd == "list" {
		us, err := st.Users()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "sb-idp:", err)
			return 1
		}
		for _, u := range us {
			state := "enabled"
			if u.Disabled {
				state = "disabled"
			}
			_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", u.ID, u.Email, state)
		}
		return 0
	}
	if (cmd != "add" && cmd != "disable") || !strings.Contains(*email, "@") || *scimURL == "" || *tokenFile == "" {
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
	tok, err := os.ReadFile(*tokenFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-idp: the SCIM token can't be read")
		return 1
	}
	sc := scim{base: strings.TrimRight(*scimURL, "/"), token: strings.TrimSpace(string(tok)), hc: &http.Client{Timeout: 30 * time.Second}}
	addr := strings.ToLower(strings.TrimSpace(*email))
	if cmd == "disable" {
		u, err := st.Disable(addr)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "sb-idp:", err)
			return 1
		}
		if err := sc.deactivate(u.ID); err != nil {
			_, _ = fmt.Fprintln(stderr, "sb-idp: disabled here, but the cell wasn't told:", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, "sb-idp: disabled", addr)
		return 0
	}
	if _, found, err := st.Find(addr); err != nil || found {
		_, _ = fmt.Fprintln(stderr, "sb-idp: that address is already a user, or the users file can't be read")
		return 2
	}
	pw, err := io.ReadAll(io.LimitReader(stdin, 1024))
	if err != nil {
		return 1
	}
	hash, err := bundled.Hash(strings.TrimRight(string(pw), "\r\n"))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-idp:", err)
		return 2
	}
	id, err := sc.create(addr)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-idp: the cell refused the user:", err)
		return 1
	}
	if err := st.Add(bundled.User{ID: id, Email: addr, Hash: hash}); err != nil {
		_, _ = fmt.Fprintln(stderr, "sb-idp:", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "sb-idp: added %s as %s\n", addr, id)
	return 0
}

type scim struct {
	base, token string
	hc          *http.Client
}

func (s scim) call(method, path string, body any) (map[string]any, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(method, s.base+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/scim+json")
	resp, err := s.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("SCIM %s %s: HTTP %d", method, path, resp.StatusCode)
	}
	return out, nil
}

func (s scim) create(email string) (string, error) {
	user, _, _ := strings.Cut(email, "@")
	out, err := s.call(http.MethodPost, "/Users", map[string]any{"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": email, "active": true, "name": map[string]string{"formatted": user}, "emails": []map[string]any{{"value": email, "primary": true}}})
	if err != nil {
		return "", err
	}
	id, _ := out["id"].(string)
	if !strings.HasPrefix(id, "usr_") {
		return "", errors.New("SCIM returned no usr_ id")
	}
	return id, nil
}

func (s scim) deactivate(id string) error {
	_, err := s.call(http.MethodPatch, "/Users/"+id, map[string]any{"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{{"op": "replace", "path": "active", "value": false}}})
	return err
}
