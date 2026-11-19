package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	auditfs "example.com/second-brain/internal/adapters/audit/localfs"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/rollout"
	"example.com/second-brain/internal/errs"
	"example.com/second-brain/internal/version"
)

// releaseState loads DIR/releases/state.json from a local control plane.
func (e *env) releaseState() (*rollout.State, string, error) {
	root, err := e.killRoot()
	if err != nil {
		return nil, "", err
	}
	p := filepath.Join(root, "releases", "state.json")
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, p, errs.New(errs.EUSAGE, "no releases yet in this control plane (%s)", p)
	}
	if err != nil {
		return nil, p, errs.New(errs.EUNAVAIL, "%v", err)
	}
	var s rollout.State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, p, errs.New(errs.EUNAVAIL, "%s is not valid: %v", p, err)
	}
	return &s, p, nil
}

func saveState(p string, s *rollout.State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// releaseAudit writes a kind: release row in the control plane's own log, before the
// state changes on disk.
func (e *env) releaseAudit(subject, state, reason string) error {
	root, err := e.killRoot()
	if err != nil {
		return err
	}
	alog := &audit.Log{Tenant: "platform", Release: func() string { return "cli-" + version.Version }, Store: auditfs.Dir{Root: root, Tenant: "platform"}, Now: time.Now}
	_, err = alog.Write(context.Background(), audit.Event{Actor: actor(), Kind: "release", Subject: subject, State: state, Reason: reason})
	return err
}

// release handles sb release status|promote|rollback (sb-release(1)).
func (e *env) release(sub string, args []string) int {
	name := "sb release " + sub
	args = e.localArgs(args)
	s, p, err := e.releaseState()
	if err != nil {
		return e.fail(name, err)
	}
	switch sub {
	case "status":
		return e.releaseStatus(s)
	case "promote":
		if len(args) != 3 || args[1] != "--to" || args[2] != "stable" {
			return e.fail(name, errs.New(errs.EUSAGE, "usage: sb release promote REL --to stable"))
		}
		if err := s.Promote(args[0], time.Now()); err != nil {
			fmt.Fprintf(e.err, "refused: %s\n", trimRefused(err))
			return 1
		}
		if err := e.releaseAudit(args[0], "promoted", "promote to stable"); err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "audit: %v; nothing changed", err))
		}
		if err := saveState(p, s); err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "%v", err))
		}
		fmt.Fprintf(e.out, "promote: %s is stable (previous %s)\n", s.Stable, s.PrevStable)
		return 0
	case "rollback":
		reason := ""
		for i := 0; i < len(args); i++ {
			if args[i] == "--reason" && i+1 < len(args) {
				reason, i = args[i+1], i+1
				continue
			}
			return e.fail(name, errs.New(errs.EUSAGE, "usage: sb release rollback [--reason TEXT]"))
		}
		if reason == "" {
			reason = "rollback"
		}
		before := s.Stable
		if _, err := s.Rollback(); err != nil {
			fmt.Fprintf(e.err, "refused: %s\n", trimRefused(err))
			return 1
		}
		if err := e.releaseAudit(before, "revoked", reason); err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "audit: %v; nothing changed", err))
		}
		if err := saveState(p, s); err != nil {
			return e.fail(name, errs.New(errs.EUNAVAIL, "%v", err))
		}
		var draining []string
		for n, t := range s.Tenants {
			if t.Draining {
				draining = append(draining, n)
			}
		}
		sort.Strings(draining)
		fmt.Fprintf(e.out, "rollback: %s revoked; stable is %s; draining until redeployed: %v\n", before, s.Stable, draining)
		return 0
	}
	return e.fail(name, errs.New(errs.EUSAGE, "unknown subcommand"))
}

func trimRefused(err error) string {
	msg := err.Error()
	const p = "rollout: refused: "
	if len(msg) > len(p) && msg[:len(p)] == p {
		return msg[len(p):]
	}
	return msg
}

func (e *env) releaseStatus(s *rollout.State) int {
	if e.g.JSON {
		return e.json(s)
	}
	fmt.Fprintf(e.out, "stable   %s\nprevious %s\n", s.Stable, s.PrevStable)
	var ids []string
	for id, r := range s.Releases {
		if r.Order > s.Releases[s.Stable].Order && r.Gated && !r.Revoked {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return s.Releases[ids[i]].Order < s.Releases[ids[j]].Order })
	for _, id := range ids {
		fmt.Fprintf(e.out, "testing  %s\n", id)
	}
	var names []string
	for n := range s.Tenants {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Fprintf(e.out, "%-18s %-8s %-24s %s\n", "tenant", "channel", "running", "lag")
	for _, n := range names {
		t := s.Tenants[n]
		lag := "-"
		if t.Channel == rollout.Stable {
			switch t.Running {
			case s.Stable:
				lag = "0"
			case s.PrevStable:
				lag = "1"
			default:
				lag = "?"
			}
		}
		if t.Draining {
			lag += " (draining)"
		}
		fmt.Fprintf(e.out, "%-18s %-8s %-24s %s\n", n, t.Channel, t.Running, lag)
	}
	return 0
}
