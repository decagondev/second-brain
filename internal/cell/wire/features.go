package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/features"
	"example.com/second-brain/internal/core/tenancy"
	"example.com/second-brain/internal/errs"
)

// Feature rollouts (sb-release(7), Feature rollouts; SB-5106): the cell keeps when it first
// ran each pilot feature and which are held, in state/features/state.json.

var featureMu sync.Mutex

func (c *Cell) featurePath() string { return filepath.Join(c.Config.State, "features", "state.json") }

// features is the rollout state now, with every pilot feature of this release seen.
func (c *Cell) features() (features.State, error) {
	featureMu.Lock()
	defer featureMu.Unlock()
	var st features.State
	b, err := os.ReadFile(c.featurePath())
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return st, err
	default:
		if err := json.Unmarshal(b, &st); err != nil {
			return st, fmt.Errorf("wire: %s: %w", c.featurePath(), err)
		}
	}
	st, changed := st.Seen(time.Now())
	if changed {
		if err := c.writeFeatures(st); err != nil {
			return st, err
		}
	}
	return st, nil
}

func (c *Cell) writeFeatures(st features.State) error {
	if err := os.MkdirAll(filepath.Dir(c.featurePath()), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(c.featurePath(), b)
}

// featureOn says feature name is on for user now. State that can't be read is off.
func (c *Cell) featureOn(ctx context.Context, user, name string) bool {
	st, err := c.features()
	if err != nil {
		return false
	}
	groups, err := c.Directory.GroupsOf(ctx, c.Config.Tenant, user)
	if err != nil {
		return false
	}
	return st.For(name, groups, time.Now())
}

// Rollouts are every feature's status now (SB-5106).
func (c *Cell) Rollouts(context.Context) ([]features.Status, error) {
	st, err := c.features()
	if err != nil {
		return nil, err
	}
	return st.All(time.Now()), nil
}

// HoldFeature holds a pilot feature, or releases it (on false), audited before it takes
// effect.
func (c *Cell) HoldFeature(ctx context.Context, by, name string, hold bool) error {
	f, ok := features.Known(name)
	if !ok {
		return fmt.Errorf("%w: %w %q", ask.ErrInvalid, features.ErrUnknown, name)
	}
	if f.Rollout != features.Pilot {
		return fmt.Errorf("%w: %s has no pilot", ask.ErrInvalid, name)
	}
	st, err := c.features()
	if err != nil {
		return err
	}
	state := "released"
	if hold {
		state = "held"
	}
	if _, err := c.Audit.Write(ctx, audit.Event{Actor: by, Kind: "config", Subject: "rollout:" + name, State: state, Reason: "feature " + state}); err != nil {
		return fmt.Errorf("%w: %v", ask.ErrUnavailable, err)
	}
	featureMu.Lock()
	defer featureMu.Unlock()
	if st.Held == nil {
		st.Held = map[string]bool{}
	}
	if hold {
		st.Held[name] = true
	} else {
		delete(st.Held, name)
	}
	return c.writeFeatures(st)
}

// holdAdmin is HoldFeature for an operator's scope.
func (c *Cell) holdAdmin(ctx context.Context, s tenancy.Scope, name string, hold bool) error {
	return c.HoldFeature(ctx, s.User(), name, hold)
}

// rolloutsCP is GET /v1/cell/rollouts and POST /v1/cell/rollouts/NAME/hold|release for sb
// rollout (SB-5106).
func (c *Cell) rolloutsCP(w http.ResponseWriter, r *http.Request) {
	if !c.guardControlPlane(w, r) {
		return
	}
	if r.Method == http.MethodPost {
		name, op := r.PathValue("feature"), r.PathValue("op")
		if op != "hold" && op != "release" {
			refuseJSON(w, errs.EINVAL, "The action is hold or release.")
			return
		}
		if err := c.HoldFeature(r.Context(), "sys_control_plane", name, op == "hold"); err != nil {
			code := errs.EUNAVAIL
			if errors.Is(err, ask.ErrInvalid) {
				code = errs.EINVAL
			}
			refuseJSON(w, code, strings.TrimPrefix(err.Error(), "ask: EINVAL: "))
			return
		}
	}
	rs, err := c.Rollouts(r.Context())
	if err != nil {
		refuseJSON(w, errs.EUNAVAIL, "The rollouts can't be read right now.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"features": rs})
}

// mayPauseLink says user may pause agent in space and has the pause link (SB-5107).
func (c *Cell) mayPauseLink(ctx context.Context, user, space, agent string) bool {
	if !c.featureOn(ctx, user, "pause-link") {
		return false
	}
	s, err := c.ScopeAs(ctx, user, "company")
	if err != nil {
		return false
	}
	via, err := c.Parts().Stewards.PauseVia(ctx, s, space, agent)
	return err == nil && (via == "" || c.featureOn(ctx, user, "branch-moderators"))
}

// rolloutLines are the admin's week's lines on rollouts (SB-5106): features in pilot, held,
// and those that went on for everyone in the week.
func (c *Cell) rolloutLines(from, to time.Time) []ask.AdminLine {
	st, err := c.features()
	if err != nil {
		return nil
	}
	var out []ask.AdminLine
	for _, s := range st.All(to) {
		switch {
		case s.State == "held":
			out = append(out, ask.AdminLine{Section: "Rollouts", Text: s.Name + " is held for the pilot group", Link: "/v1/admin/rollouts"})
		case s.State == features.Pilot:
			out = append(out, ask.AdminLine{Section: "Rollouts", Text: s.Name + " is in pilot until " + s.OnAt.Format(time.DateOnly), Link: "/v1/admin/rollouts"})
		case s.Rollout == features.Pilot && !s.Since.IsZero():
			if on := s.Since.AddDate(0, 0, features.PilotDays); !on.Before(from) && on.Before(to) {
				out = append(out, ask.AdminLine{Section: "Rollouts", Text: s.Name + " went on for everyone", Link: "/v1/admin/rollouts"})
			}
		}
	}
	return out
}
