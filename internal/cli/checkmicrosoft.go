package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/msgraph"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/errs"
)

// MicrosoftRunbook is where each check-microsoft fix is explained.
const MicrosoftRunbook = "docs/runbooks/microsoft.md"

// msLine is one check-microsoft result; Fix is the runbook section for a failure.
type msLine struct {
	OK     bool   `json:"ok"`
	Check  string `json:"check"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

const checkMicrosoftUse = "usage: sb selfhost check-microsoft --config FILE [--vault DIR] [--mailbox ADDR]... [--probe ADDR]"

// selfhostCheckMicrosoft checks a cell's Microsoft 365 setup (SB-6003): the app
// registration and its token, that every application permission only reads, that mail is
// limited to the shared mailboxes the manifest names (a --probe mailbox, such as a person's
// own, must be refused), that each named drive answers, and which of the tenant's labels
// labels.microsoft doesn't map. It reads only and prints no secret.
func (e *env) selfhostCheckMicrosoft(args []string) int {
	const cmd = "sb selfhost check-microsoft"
	args = e.localArgs(args)
	opt := map[string]string{}
	var boxes []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--mailbox" && i+1 < len(args):
			boxes = append(boxes, args[i+1])
			i++
		case (a == "--config" || a == "--vault" || a == "--probe") && i+1 < len(args):
			opt[a] = args[i+1]
			i++
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, checkMicrosoftUse))
		}
	}
	if opt["--config"] == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, checkMicrosoftUse))
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
	if src, err := os.ReadFile(cfg.Manifest); err == nil {
		if t, ds := manifest.Check(src); len(ds) == 0 {
			for _, name := range sortedKeys(t.Spaces) {
				if mb := t.Spaces[name].Mailbox; mb != "" {
					boxes = append(boxes, mb)
				}
			}
		}
	}
	g, _ := wire.GraphFor(cfg, vault, &http.Client{Timeout: 30 * time.Second})
	lines := checkMicrosoft(context.Background(), g, cfg, boxes, opt["--probe"])
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
			v := "ok"
			if !l.OK {
				v = "fail"
			}
			fmt.Fprintf(e.out, "%-4s %-12s %s\n", v, l.Check, l.Detail)
			if l.Fix != "" {
				fmt.Fprintf(e.out, "     fix: %s#%s\n", MicrosoftRunbook, l.Fix)
			}
		}
	}
	if failed > 0 {
		return 1
	}
	return 0
}

func sortedKeys[V any](m map[string]V) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// checkMicrosoft runs the checks; a check it can't run fails (Unknown is refusal).
func checkMicrosoft(ctx context.Context, g *msgraph.Client, cfg wire.Config, boxes []string, probe string) []msLine {
	var out []msLine
	add := func(ok bool, check, detail, fix string) {
		if ok {
			fix = ""
		}
		out = append(out, msLine{OK: ok, Check: check, Detail: detail, Fix: fix})
	}
	if _, err := g.Token(ctx); err != nil {
		add(false, "token", "the app registration's credentials get no token", "app-registration")
		return out
	}
	add(true, "token", "the app registration signs in", "")
	granted, err := g.Granted(ctx)
	switch {
	case err != nil:
		add(false, "permissions", "the app registration's permissions can't be read", "app-registration")
	case len(granted) == 0:
		add(false, "permissions", "the app registration holds no application permission", "app-registration")
	default:
		var writes []string
		for _, p := range granted {
			if !msgraph.ReadOnly(p) {
				writes = append(writes, p)
			}
		}
		if len(writes) > 0 {
			add(false, "permissions", "permissions that don't only read: "+strings.Join(writes, ", ")+"; remove them", "write-permission")
		} else {
			add(true, "permissions", fmt.Sprintf("%d application permissions, each read-only", len(granted)), "")
		}
	}
	hasMail := false
	for _, p := range granted {
		hasMail = hasMail || p == "Mail.Read"
	}
	if hasMail || len(boxes) > 0 {
		for _, mb := range boxes {
			ok, err := g.CanRead(ctx, mb)
			add(ok && err == nil, "mailbox", "the shared mailbox "+safeWord(mb)+map[bool]string{true: " reads", false: " can't be read"}[ok && err == nil], "shared-mailbox")
		}
		switch probe {
		case "":
			add(false, "mail policy", "no --probe mailbox: the access policy limiting mail to the named mailboxes can't be confirmed", "mail-access-policy")
		default:
			ok, err := g.CanRead(ctx, probe)
			switch {
			case err != nil:
				add(false, "mail policy", "the probe mailbox gave no answer", "mail-access-policy")
			case ok:
				add(false, "mail policy", "a mailbox not named can be read: limit Mail.Read with an application access policy", "mail-access-policy")
			default:
				add(true, "mail policy", "a mailbox not named is refused", "")
			}
		}
	}
	if len(cfg.Microsoft.Drives) == 0 {
		add(false, "drives", "microsoft.drives names no drive", "sites-and-drives")
	}
	for _, d := range cfg.Microsoft.Drives {
		err := g.DriveAnswers(ctx, d)
		add(err == nil, "drives", "the drive "+safeWord(d)+map[bool]string{true: " answers", false: " doesn't answer"}[err == nil], "sites-and-drives")
	}
	ids, err := g.LabelIDs(ctx)
	if err != nil {
		add(false, "labels", "the tenant's sensitivity labels can't be read: grant InformationProtectionPolicy.Read.All", "label-mapping")
		return out
	}
	var un []string
	for _, id := range ids {
		if _, ok := cfg.Labels.Microsoft[id]; !ok {
			un = append(un, safeWord(id))
		}
	}
	if len(un) > 0 {
		add(false, "labels", fmt.Sprintf("%d of %d labels aren't mapped and count as restricted: %s", len(un), len(ids), strings.Join(un, ", ")), "label-mapping")
	} else {
		add(true, "labels", fmt.Sprintf("all %d labels are mapped", len(ids)), "")
	}
	return out
}
