package wire

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/chatapp/mmapp"
	"example.com/second-brain/internal/adapters/chatapp/slackapp"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/notify"
)

// stewardFacts are who answers each space's questions (ask.SpaceFacts), from m.
func stewardFacts(m manifest.Tenant) ask.SpaceFacts {
	return func(space string) ([]string, string, bool) {
		if space == "company" {
			return []string{m.QuestionsTo}, "", m.QuestionsTo != ""
		}
		sp, ok := m.Spaces[strings.TrimPrefix(space, "spc_")]
		return sp.Stewards, sp.Label, ok && len(sp.Stewards) > 0
	}
}

// Notifier is the chat bot notifications go through (sb-ask(3), Notifications): the
// Mattermost bot when the cell has one, else the Slack bot, else none.
func (c *Cell) Notifier() notify.Sender {
	client := &http.Client{Timeout: 10 * time.Second}
	if mm := c.Config.Mattermost; mm.Base != "" {
		return mmapp.DM{Base: mm.Base, Token: slackField(c.Secret("mattermost"), "bot_token"), Client: client}
	}
	if cc, ok := c.Config.Connectors["chat"]; ok {
		return slackapp.DM{Base: cc.Base, Token: slackField(c.Secret("connectors/chat"), "bot_token"), Client: client}
	}
	return nil
}

// NotifyJob is the cell's notify job, with the applied manifest.
func (c *Cell) NotifyJob(send notify.Sender) notify.Job {
	m := c.Manifest()
	return notify.Job{Tenant: c.Config.Tenant, Base: c.Config.BaseURL, Questions: c.Traces, Facts: stewardFacts(m),
		Dir: c.Directory, Prefs: c.Traces, Send: send, Clock: notifyClock{filepath.Join(c.Config.State, "notify", "last")},
		Now: time.Now, Lookback: time.Duration(m.RetentionDays) * 24 * time.Hour,
		Failed: func(kind string) {
			c.Metrics.Add("sb_notify_failed_total", "Notifications that couldn't be sent, by kind (sb-ask(3)).", map[string]string{"kind": kind}, 1)
		}}
}

// notifyClock keeps the notify job's last run in a file under the cell's state.
type notifyClock struct{ path string }

func (n notifyClock) Last(context.Context) (time.Time, bool, error) {
	b, err := os.ReadFile(n.path)
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(b)))
	return t, err == nil, err
}

func (n notifyClock) Mark(_ context.Context, t time.Time) error {
	if err := os.MkdirAll(filepath.Dir(n.path), 0o700); err != nil {
		return err
	}
	tmp := n.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(t.UTC().Format(time.RFC3339Nano)+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, n.path)
}
