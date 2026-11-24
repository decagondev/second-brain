package cli

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/secrets/localvault"
	"example.com/second-brain/internal/adapters/smtpmail"
	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/errs"
)

// Checks that make a box easier to operate (sb-selfhost(7), Checking the settings;
// SB-2601 to SB-2604). None of them prints a secret.

// selfhostCheckEmail checks the company's mail server (SB-2601): `sb selfhost check-email
// HOST:PORT [--ca FILE]`. It answers, offers STARTTLS and shows a certificate the system's
// roots (and --ca) trust for HOST. It sends nothing and signs in to nothing.
func (e *env) selfhostCheckEmail(args []string) int {
	const cmd = "sb selfhost check-email"
	const use = "usage: sb selfhost check-email HOST:PORT [--ca FILE]"
	args = e.localArgs(args)
	var server, ca string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--ca" && i+1 < len(args):
			ca = args[i+1]
			i++
		case server == "" && a != "" && a[0] != '-':
			server = a
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, use))
		}
	}
	if server == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, use))
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if ca != "" {
		b, err := os.ReadFile(ca)
		if err != nil || !roots.AppendCertsFromPEM(b) {
			return e.fail(cmd, errs.New(errs.EINVAL, "%s holds no PEM certificate", ca))
		}
	}
	if err := smtpmail.Probe(context.Background(), server, roots); err != nil {
		if errors.Is(err, smtpmail.ErrNoTLS) {
			return e.fail(cmd, errs.New(errs.EINVAL, "%s doesn't offer STARTTLS; the cell never sends in the clear", server))
		}
		return e.fail(cmd, errs.New(errs.EUNAVAIL, "%s: %v", server, err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"server": server, "starttls": true})
	}
	fmt.Fprintf(e.out, "email: %s answers and offers STARTTLS with a trusted certificate\n", server)
	return 0
}

// selfhostCheckNotify sends the fixed test notification through the cell's real channel
// (SB-2602): `sb selfhost check-notify --to EMAIL (--cell URL | --tenant T)`, as an operator
// (SB_TOKEN). At most once a minute per address.
func (e *env) selfhostCheckNotify(args []string) int {
	const cmd = "sb selfhost check-notify"
	const use = "usage: sb selfhost check-notify --to EMAIL (--cell URL | --tenant T)"
	args = e.localArgs(args)
	var o cellOpts
	var to string
	for i := 0; i < len(args); i++ {
		if took, err := o.take(args, &i); took {
			if err != nil {
				return e.fail(cmd, err)
			}
			continue
		}
		if args[i] == "--to" && i+1 < len(args) {
			to = args[i+1]
			i++
			continue
		}
		return e.fail(cmd, errs.New(errs.EUSAGE, use))
	}
	if to == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, use))
	}
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(cmd, err)
	}
	body, _ := json.Marshal(map[string]string{"to": to})
	raw, err := cellDo(http.MethodPost, cellURL, "/v1/admin/notifications/test", "application/json", body, time.Minute)
	if err != nil {
		return e.fail(cmd, err)
	}
	var res struct {
		Sent bool   `json:"sent"`
		Via  string `json:"via"`
	}
	if err := json.Unmarshal(raw, &res); err != nil || res.Via == "" {
		return e.fail(cmd, errs.New(errs.EUNAVAIL, "the cell's answer isn't a test result"))
	}
	if e.g.JSON {
		return e.json(res)
	}
	if !res.Sent {
		fmt.Fprintf(e.out, "notify: %s doesn't know %s; check the person's email there\n", res.Via, to)
		return 1
	}
	fmt.Fprintf(e.out, "notify: %s took the test for %s\n", res.Via, to)
	return 0
}

// Scopes the Slack app needs (sb-ask(3), Slack), each with what it is for.
var slackScopes = []struct{ scope, why string }{
	{"commands", "/ask"},
	{"chat:write", "notifications"},
	{"im:write", "notifications"},
	{"users:read.email", "finding people by their email"},
}

// selfhostCheckChat asks the chat what the cell's bot may do (SB-2603): `sb selfhost
// check-chat --slack TOKENFILE [--api URL]` or `--mattermost URL --token-file FILE [--team
// ID]`. A token file holds the token, or the vault's JSON with bot_token. The token is
// never printed.
func (e *env) selfhostCheckChat(args []string) int {
	const cmd = "sb selfhost check-chat"
	const use = "usage: sb selfhost check-chat --slack TOKENFILE [--api URL] | --mattermost URL --token-file FILE [--team ID]"
	args = e.localArgs(args)
	opt := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--slack", "--api", "--mattermost", "--token-file", "--team":
			if i+1 >= len(args) {
				return e.fail(cmd, errs.New(errs.EUSAGE, use))
			}
			opt[a] = args[i+1]
			i++
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, use))
		}
	}
	var lines []string
	var err error
	switch {
	case opt["--slack"] != "" && opt["--mattermost"] == "":
		lines, err = checkSlack(opt["--slack"], opt["--api"])
	case opt["--mattermost"] != "" && opt["--token-file"] != "" && opt["--slack"] == "":
		lines, err = checkMattermost(opt["--mattermost"], opt["--token-file"], opt["--team"])
	default:
		return e.fail(cmd, errs.New(errs.EUSAGE, use))
	}
	if err != nil {
		return e.fail(cmd, err)
	}
	missing := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "missing") {
			missing++
		}
	}
	if e.g.JSON {
		_ = e.json(map[string]any{"lines": lines, "missing": missing})
	} else {
		for _, l := range lines {
			fmt.Fprintln(e.out, "chat: "+l)
		}
	}
	if missing > 0 {
		return 1
	}
	return 0
}

// tokenFrom reads a bot token from a file: the token, or {"bot_token": "…"}.
func tokenFrom(file string) (string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return "", errs.New(errs.EINVAL, "can't read %s", file)
	}
	s := strings.TrimSpace(string(b))
	if strings.HasPrefix(s, "{") {
		var v struct {
			BotToken string `json:"bot_token"`
		}
		if json.Unmarshal([]byte(s), &v) != nil || v.BotToken == "" {
			return "", errs.New(errs.EINVAL, "%s is JSON without bot_token", file)
		}
		s = v.BotToken
	}
	if s == "" || strings.ContainsAny(s, " \r\n") {
		return "", errs.New(errs.EINVAL, "%s holds no token", file)
	}
	return s, nil
}

var chatClient = &http.Client{Timeout: 20 * time.Second}

// chatResp is a chat API's answer, its body already read and closed.
type chatResp struct {
	StatusCode int
	Status     string
	Header     http.Header
}

func chatGet(method, url, token string, body io.Reader) (chatResp, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return chatResp{}, nil, errs.New(errs.EUSAGE, "bad URL %s", url)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := chatClient.Do(req)
	if err != nil {
		return chatResp{}, nil, errs.New(errs.EUNAVAIL, "%s didn't answer", strings.SplitN(url, "/api/", 2)[0])
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return chatResp{resp.StatusCode, resp.Status, resp.Header}, b, nil
}

func checkSlack(file, api string) ([]string, error) {
	token, err := tokenFrom(file)
	if err != nil {
		return nil, err
	}
	if api == "" {
		api = "https://slack.com/api"
	}
	resp, b, err := chatGet(http.MethodPost, strings.TrimSuffix(api, "/")+"/auth.test", token, bytes.NewReader(nil))
	if err != nil {
		return nil, err
	}
	var v struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Team  string `json:"team"`
		User  string `json:"user"`
	}
	if json.Unmarshal(b, &v) != nil || !v.OK {
		if v.Error == "" {
			v.Error = resp.Status
		}
		return nil, errs.New(errs.EAUTH, "Slack refused the token (%s); copy the Bot User OAuth Token from the app's OAuth & Permissions page", safeWord(v.Error))
	}
	have := map[string]bool{}
	for _, s := range strings.Split(resp.Header.Get("X-OAuth-Scopes"), ",") {
		have[strings.TrimSpace(s)] = true
	}
	lines := []string{fmt.Sprintf("Slack: the bot %s in %s", safeWord(v.User), safeWord(v.Team))}
	for _, s := range slackScopes {
		if have[s.scope] {
			lines = append(lines, "ok "+s.scope+" ("+s.why+")")
		} else {
			lines = append(lines, "missing "+s.scope+" ("+s.why+"): add it under OAuth & Permissions, Bot Token Scopes, at api.slack.com/apps, then reinstall the app")
		}
	}
	return lines, nil
}

func checkMattermost(base, file, team string) ([]string, error) {
	token, err := tokenFrom(file)
	if err != nil {
		return nil, err
	}
	base = strings.TrimSuffix(base, "/")
	resp, b, err := chatGet(http.MethodGet, base+"/api/v4/users/me", token, nil)
	if err != nil {
		return nil, err
	}
	var me struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		IsBot    bool   `json:"is_bot"`
	}
	if resp.StatusCode != http.StatusOK || json.Unmarshal(b, &me) != nil || me.ID == "" {
		return nil, errs.New(errs.EAUTH, "Mattermost refused the token (%s); make one under Integrations, Bot Accounts", resp.Status)
	}
	lines := []string{"Mattermost: the account " + safeWord(me.Username)}
	if me.IsBot {
		lines = append(lines, "ok a bot account")
	} else {
		lines = append(lines, "ok a user account (a bot account under Integrations, Bot Accounts is better)")
	}
	_, b, err = chatGet(http.MethodGet, base+"/api/v4/users/me/teams", token, nil)
	if err != nil {
		return nil, err
	}
	var teams []struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &teams)
	in := false
	for _, t := range teams {
		in = in || team == "" || t.ID == team
	}
	if in && len(teams) > 0 {
		lines = append(lines, "ok a member of the team (slash commands)")
	} else {
		lines = append(lines, "missing team membership (slash commands): add the bot to the team, under the team's Main Menu, Add Members")
	}
	_, b, err = chatGet(http.MethodGet, base+"/api/v4/users?per_page=20", token, nil)
	if err != nil {
		return nil, err
	}
	var users []struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	_ = json.Unmarshal(b, &users)
	shown := false
	for _, u := range users {
		shown = shown || (u.ID != me.ID && u.Email != "")
	}
	if shown {
		lines = append(lines, "ok people's emails are shown (finding people)")
	} else {
		lines = append(lines, "missing people's emails (finding people): turn on Show Email Address under System Console, Site Configuration, Users and Teams")
	}
	return lines, nil
}

var plainWord = regexp.MustCompile(`^[A-Za-z0-9 ._@:-]{1,80}$`)

// safeWord is a name from the chat, or "?" when it isn't plainly one.
func safeWord(s string) string {
	if plainWord.MatchString(s) {
		return s
	}
	return "?"
}

// cellCheckConfig checks a cell's configuration before a restart (SB-2604): `sb cell
// check-config FILE [--vault DIR]...`. It runs the services' own check, then confirms each
// secret the configuration needs is in a vault with owner-only permissions, without
// reading it. A cell whose services each hold their own vault (sb-selfhost(7)) names every
// one: a secret may be in any of them, and a copy others can read anywhere is refused. A
// failure names the line at fault.
func (e *env) cellCheckConfig(args []string) int {
	const cmd = "sb cell check-config"
	const use = "usage: sb cell check-config FILE [--vault DIR]..."
	args = e.localArgs(args)
	var file string
	var vaults []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--vault" && i+1 < len(args):
			vaults = append(vaults, args[i+1])
			i++
		case file == "" && a != "" && a[0] != '-':
			file = a
		default:
			return e.fail(cmd, errs.New(errs.EUSAGE, use))
		}
	}
	if file == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, use))
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", file))
	}
	cfg, err := wire.Parse(b)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%s%s: %v", file, lineOf(b, err), err))
	}
	if len(vaults) == 0 {
		vaults = []string{cfg.Vault}
	}
	required, optional := cfg.Secrets()
	var problems, ok []string
	check := func(p string, need bool) {
		// Found owner-only in one vault is enough; exposed in any is not.
		err := os.ErrNotExist
		for _, v := range vaults {
			e := localvault.Dir{Root: v}.Check("vault://tenants/" + cfg.Tenant + "/" + p)
			switch {
			case errors.Is(e, localvault.ErrExposed):
				err = e
			case errors.Is(err, localvault.ErrExposed):
			case e == nil:
				err = nil
			case err != nil && !errors.Is(e, os.ErrNotExist):
				err = e
			}
		}
		switch {
		case err == nil:
			ok = append(ok, p)
		case errors.Is(err, localvault.ErrExposed):
			problems = append(problems, fmt.Sprintf("the secret %s is readable by others: chmod 600 it", p))
		case errors.Is(err, os.ErrNotExist) && !need:
		default:
			problems = append(problems, fmt.Sprintf("the secret %s isn't in the vault", p))
		}
	}
	for _, p := range required {
		check(p, true)
	}
	for _, p := range optional {
		check(p, false)
	}
	sort.Strings(ok)
	if len(problems) > 0 {
		return e.fail(cmd, errs.New(errs.EINVAL, "%s: %s", file, strings.Join(problems, "; ")))
	}
	if e.g.JSON {
		return e.json(map[string]any{"config": "ok", "secrets": ok})
	}
	fmt.Fprintf(e.out, "config: %s is right, and its %d secrets are in the vault, owner-only\n", file, len(ok))
	return 0
}

var quoted = regexp.MustCompile(`"([^"]+)"`)

// lineOf names the line of b an error from wire.Parse is about: a JSON syntax error's
// offset, or the first line holding the field or value it quotes.
func lineOf(b []byte, err error) string {
	line := func(off int) string {
		if off < 0 || off > len(b) {
			return ""
		}
		return fmt.Sprintf(", line %d", bytes.Count(b[:off], []byte("\n"))+1)
	}
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return line(int(se.Offset))
	}
	var te *json.UnmarshalTypeError
	if errors.As(err, &te) {
		return line(int(te.Offset))
	}
	if m := quoted.FindStringSubmatch(err.Error()); m != nil {
		if i := bytes.Index(b, []byte(`"`+m[1]+`"`)); i >= 0 {
			return line(i)
		}
	}
	return ""
}
