package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/atlassian"
	"example.com/second-brain/internal/adapters/secrets/localvault"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/errs"
)

// AtlassianRunbook is where each check-atlassian fix is explained.
const AtlassianRunbook = "docs/runbooks/atlassian.md"

const checkAtlassianUse = "usage: sb selfhost check-atlassian --config FILE [--vault DIR]"

// selfhostCheckAtlassian checks a cell's Confluence and Jira setup (SB-6203): the API token
// signs in, its account holds no write permission in Jira (site-wide or in a named
// project), each named Confluence space and Jira project answers, and the account's groups
// resolve. It reads only and prints no secret.
func (e *env) selfhostCheckAtlassian(args []string) int {
	const cmd = "sb selfhost check-atlassian"
	args = e.localArgs(args)
	opt := map[string]string{}
	for i := 0; i < len(args); i++ {
		if (args[i] != "--config" && args[i] != "--vault") || i+1 >= len(args) {
			return e.fail(cmd, errs.New(errs.EUSAGE, checkAtlassianUse))
		}
		opt[args[i]] = args[i+1]
		i++
	}
	if opt["--config"] == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, checkAtlassianUse))
	}
	b, err := os.ReadFile(opt["--config"])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", opt["--config"]))
	}
	cfg, err := wire.Parse(b)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%s: %v", opt["--config"], err))
	}
	if cfg.Atlassian.Base == "" {
		return e.fail(cmd, errs.New(errs.EINVAL, "the configuration names no atlassian.base"))
	}
	vault := opt["--vault"]
	if vault == "" {
		vault = cfg.Vault
	}
	v := localvault.Dir{Root: vault}
	cl := &atlassian.Client{Base: cfg.Atlassian.Base, HTTP: &http.Client{Timeout: 30 * time.Second},
		Creds: func(ctx context.Context) (string, error) {
			return v.Get(ctx, "vault://tenants/"+cfg.Tenant+"/atlassian")
		}}
	lines := checkAtlassian(context.Background(), cl, cfg)
	failed := 0
	for _, l := range lines {
		if !l.OK {
			failed++
		}
	}
	if e.g.JSON {
		_ = e.json(map[string]any{"checks": lines, "failed": failed})
	} else {
		for _, l := range lines {
			s := "ok"
			if !l.OK {
				s = "fail"
			}
			fmt.Fprintf(e.out, "%-4s %-12s %s\n", s, l.Check, l.Detail)
			if l.Fix != "" {
				fmt.Fprintf(e.out, "     fix: %s#%s\n", AtlassianRunbook, l.Fix)
			}
		}
	}
	if failed > 0 {
		return 1
	}
	return 0
}

func checkAtlassian(ctx context.Context, cl *atlassian.Client, cfg wire.Config) []msLine {
	var out []msLine
	add := func(ok bool, check, detail, fix string) {
		if ok {
			fix = ""
		}
		out = append(out, msLine{OK: ok, Check: check, Detail: detail, Fix: fix})
	}
	me, err := cl.Myself(ctx)
	if err != nil {
		add(false, "token", "the API token signs in to nothing", "api-token")
		return out
	}
	add(true, "token", "the API token signs in", "")
	for _, p := range append([]string{""}, cfg.Atlassian.Projects...) {
		w, err := cl.Writes(ctx, p)
		where := "site-wide"
		if p != "" {
			where = "in " + safeWord(p)
		}
		switch {
		case err != nil:
			add(false, "permissions", "the account's permissions "+where+" can't be read", "api-token")
		case len(w) > 0:
			add(false, "permissions", "the account can write "+where+": "+strings.Join(w, ", ")+"; use a read-only account", "read-only-account")
		default:
			add(true, "permissions", "the account only reads "+where, "")
		}
	}
	for _, s := range cfg.Atlassian.Spaces {
		err := cl.SpaceAnswers(ctx, s)
		add(err == nil, "spaces", "the Confluence space "+safeWord(s)+map[bool]string{true: " answers", false: " doesn't answer"}[err == nil], "spaces-and-projects")
	}
	for _, p := range cfg.Atlassian.Projects {
		err := cl.ProjectAnswers(ctx, p)
		add(err == nil, "projects", "the Jira project "+safeWord(p)+map[bool]string{true: " answers", false: " doesn't answer"}[err == nil], "spaces-and-projects")
	}
	if _, err := (&atlassian.Directory{Client: cl}).GroupsOf(ctx, me); err != nil {
		add(false, "groups", "the account's groups don't resolve: the live check can't place people", "groups")
	} else {
		add(true, "groups", "groups resolve", "")
	}
	return out
}
