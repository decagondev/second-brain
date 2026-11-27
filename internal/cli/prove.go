package cli

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/teamsapp"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	"example.com/second-brain/internal/adapters/smtpmail"
	"example.com/second-brain/internal/cell/prove"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/proof"
	"example.com/second-brain/internal/errs"
)

var proveFlags = map[string]bool{"--config": true, "--vault": true, "--out": true, "--person": true, "--probe": true, "--facts": true, "--ca": true, "--verify": true, "--key": true}

const proveUse = `usage: sb prove TICKET --config FILE [--vault DIR] [--out DIR] [--person EMAIL] [--mailbox ADDR]... [--probe ADDR] [--facts FILE] [--ca FILE]
       sb prove --verify FILE [--key KEY]
       sb prove --list`

// prove runs a ticket's acceptance criteria against the real system the cell's
// configuration names and writes a signed proof (SB-6001; sb(1), sb prove). It reads only.
// The proof holds pass or fail, counts, timings and hashed ids, never text or names.
func (e *env) prove(args []string) int {
	const cmd = "sb prove"
	args = e.localArgs(args)
	opt := map[string]string{}
	var ticket string
	var mailboxes []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--list":
			opt[a] = "y"
		case a == "--mailbox" && i+1 < len(args):
			mailboxes = append(mailboxes, args[i+1])
			i++
		case proveFlags[a] && i+1 < len(args):
			opt[a] = args[i+1]
			i++
		case ticket == "" && strings.HasPrefix(a, "SB-"):
			ticket = a
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, proveUse))
		}
	}
	switch {
	case opt["--list"] != "":
		for _, t := range prove.Tickets() {
			fmt.Fprintln(e.out, t)
		}
		return 0
	case opt["--verify"] != "":
		b, err := os.ReadFile(opt["--verify"])
		if err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", opt["--verify"]))
		}
		s, err := proof.Verify(b, opt["--key"])
		if err != nil {
			return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
		}
		if e.g.JSON {
			return e.json(map[string]any{"ticket": s.Proof.Ticket, "passed": s.Proof.Passed, "at": s.Proof.At, "key": s.Key})
		}
		fmt.Fprintf(e.out, "proof: %s at %s verifies (key %s); passed: %v\n", s.Proof.Ticket, s.Proof.At, s.Key, s.Proof.Passed)
		return 0
	case ticket == "" || opt["--config"] == "":
		return e.fail(cmd, errs.New(errs.EUSAGE, proveUse))
	}
	if _, ok := prove.Runners[ticket]; !ok {
		return e.fail(cmd, errs.New(errs.EINVAL, "sb prove doesn't cover %s; see sb prove --list", ticket))
	}
	b, err := os.ReadFile(opt["--config"])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", opt["--config"]))
	}
	cfg, err := wire.Parse(b)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%s: %v", opt["--config"], err))
	}
	vault := opt["--vault"]
	if vault == "" {
		vault = cfg.Vault
	}
	ctx := context.Background()
	env, err := proveEnv(cfg, vault, opt, mailboxes)
	if err != nil {
		return e.fail(cmd, err)
	}
	p, err := prove.Run(ctx, ticket, env)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	seed, err := localvault.Dir{Root: vault}.Get(ctx, "vault://tenants/"+cfg.Tenant+"/audit/anchor-seed")
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "the anchor key isn't in the vault: a proof is signed with it"))
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(seed))
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "the anchor seed isn't base64"))
	}
	sg, err := audit.NewSigner(cfg.Tenant, raw)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	signed, err := proof.Seal(p, sg, time.Now())
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	dir := opt["--out"]
	if dir == "" {
		dir = filepath.Join(cfg.State, "proofs")
	}
	out, err := writeProof(dir, signed)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	code := 0
	if !p.Passed {
		code = 1
	}
	if e.g.JSON {
		_ = e.json(map[string]any{"ticket": ticket, "passed": p.Passed, "file": out, "criteria": p.Criteria})
		return code
	}
	for _, c := range p.Criteria {
		v := "pass"
		if !c.Pass {
			v = "fail"
		}
		fmt.Fprintf(e.out, "%-5s %s%s\n", v, c.Name, countLine(c.Counts))
	}
	fmt.Fprintf(e.out, "prove: %s passed: %v; signed proof in %s\n", ticket, p.Passed, out)
	return code
}

func countLine(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	var parts []string
	for _, k := range ks {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return " (" + strings.Join(parts, " ") + ")"
}

func writeProof(dir string, s proof.Signed) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, proof.FileName(s.Proof))
	return p, os.WriteFile(p, b, 0o600)
}

// proveEnv builds what the runners may use from the cell's configuration.
func proveEnv(cfg wire.Config, vault string, opt map[string]string, mailboxes []string) (prove.Env, error) {
	hc := &http.Client{Timeout: 30 * time.Second}
	env := prove.Env{Tenant: cfg.Tenant, Mailboxes: mailboxes, Probe: opt["--probe"], Person: opt["--person"], Labels: cfg.Labels.Microsoft}
	if cfg.Microsoft.Base != "" || len(cfg.Microsoft.Drives) > 0 || cfg.Microsoft.Domain != "" {
		env.Graph, env.SharePoint = wire.GraphFor(cfg, vault, hc)
	}
	if cfg.Teams.AppID != "" {
		app := &teamsapp.App{AppID: cfg.Teams.AppID, OpenID: cfg.Teams.OpenID, HTTP: hc}
		env.TeamsKeys = app.SigningKeys
	}
	if cfg.Mattermost.Base != "" {
		tok := filepath.Join(vault, "vault", "tenants", cfg.Tenant, "mattermost")
		env.Chat = func() ([]string, error) { return checkMattermost(cfg.Mattermost.Base, tok, cfg.Mattermost.TeamID) }
	}
	if cfg.Email.Server != "" {
		roots, err := x509.SystemCertPool()
		if err != nil {
			roots = x509.NewCertPool()
		}
		if ca := opt["--ca"]; ca != "" {
			b, err := os.ReadFile(ca)
			if err != nil || !roots.AppendCertsFromPEM(b) {
				return env, errs.New(errs.EINVAL, "%s holds no PEM certificate", ca)
			}
		}
		env.Email = func(ctx context.Context) error { return smtpmail.Probe(ctx, cfg.Email.Server, roots) }
	}
	if f := opt["--facts"]; f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return env, errs.New(errs.EINVAL, "can't read %s", f)
		}
		env.Facts = b
	}
	return env, nil
}
