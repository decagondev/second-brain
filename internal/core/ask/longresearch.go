package ask

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/budget"
	"example.com/second-brain/internal/core/tenancy"
)

// Long research with checkpoints (sb-ask(3), Long research; SB-4903): a long report shows its
// plan (its sub-questions, the space it searches, its documents, an estimate and its
// ceiling) and runs nothing until its owner presses Run. It then runs in at most MaxStages
// stages, one a notify run, for at most MaxLongRun. After each stage but the last its owner
// is told, and may steer (add or drop sub-questions not yet asked), stop it with what it
// has, or let it go on; with no answer for CheckpointWait it goes on. It stops at its
// ceiling with what it has. Every stage runs as its owner as they are now, and the sections
// of earlier stages are checked again as them first, so a document they lost is dropped
// at the next stage (LAW-26).

// StateCheckpoint is a long run waiting for its owner, as the notify job is told it.
const StateCheckpoint = "checkpoint"

// The long research limits.
const (
	MaxStages      = 6
	MaxLongRun     = 4 * time.Hour
	CheckpointWait = 30 * time.Minute
	// DefaultLongUSD is a long run's ceiling when the tenant sets no research_usd.
	DefaultLongUSD = 5.0
	// StepEstimateUSD is the planning figure for one sub-question; the ceiling, not the
	// estimate, is what stops a run.
	StepEstimateUSD = 0.10
)

// LongRun is a long report's stages, as the cell keeps them.
type LongRun struct {
	PerStage    int        `json:"per_stage"`
	Stage       int        `json:"stage"` // stages done
	Asked       int        `json:"asked"` // sub-questions asked
	EstimateUSD float64    `json:"estimate_usd"`
	LimitUSD    float64    `json:"limit_usd"`
	Started     time.Time  `json:"started,omitempty"`
	Checkpoint  *time.Time `json:"checkpoint,omitempty"` // waiting on its owner since
	Dropped     int        `json:"dropped,omitempty"`    // claims dropped by a later stage's check
}

// LongView is a long run as its owner sees it.
type LongView struct {
	Stage       int        `json:"stage"`
	Stages      int        `json:"stages"`
	Asked       int        `json:"asked"`
	EstimateUSD float64    `json:"estimate_usd"`
	LimitUSD    float64    `json:"limit_usd"`
	Checkpoint  *time.Time `json:"checkpoint,omitempty"`
	GoesOnAt    *time.Time `json:"goes_on_at,omitempty"`
}

// Steer is an owner's change to a long run at a checkpoint: sub-questions to add, and the
// indexes (from 0, in steps) of ones not yet asked to drop.
type Steer struct {
	Add  []string `json:"add"`
	Drop []int    `json:"drop"`
}

func (l Legwork) longUSD() float64 {
	if l.ResearchUSD != nil {
		if v := l.ResearchUSD(); v > 0 {
			return v
		}
	}
	return DefaultLongUSD
}

func (l Legwork) longPlan(steps int) *LongRun {
	return &LongRun{PerStage: perStage(steps), EstimateUSD: float64(steps) * StepEstimateUSD, LimitUSD: l.longUSD()}
}

func perStage(steps int) int {
	return max(1, (steps+MaxStages-1)/MaxStages)
}

func stagesOf(r Research) int {
	if r.Long == nil {
		return 0
	}
	left := len(r.Steps) - r.Long.Asked
	return r.Long.Stage + (max(0, left)+r.Long.PerStage-1)/r.Long.PerStage
}

func longView(r Research) *LongView {
	if r.Long == nil {
		return nil
	}
	v := &LongView{Stage: r.Long.Stage, Stages: stagesOf(r), Asked: r.Long.Asked, EstimateUSD: r.Long.EstimateUSD, LimitUSD: r.Long.LimitUSD, Checkpoint: r.Long.Checkpoint}
	if r.Long.Checkpoint != nil {
		t := r.Long.Checkpoint.Add(CheckpointWait)
		v.GoesOnAt = &t
	}
	return v
}

// waiting is a long run at a checkpoint its owner may still answer.
func waiting(r Research, now time.Time) bool {
	return r.Long != nil && r.Long.Checkpoint != nil && now.Before(r.Long.Checkpoint.Add(CheckpointWait))
}

// atCheckpoint is a long run of s's own that waits at a checkpoint.
func (l Legwork) atCheckpoint(ctx context.Context, s tenancy.Scope, id string) (Research, error) {
	r, err := l.own(ctx, s, id)
	if err != nil {
		return Research{}, err
	}
	if r.Long == nil || r.State != StateQueued || r.Long.Checkpoint == nil {
		return Research{}, fmt.Errorf("%w: only a long report at a checkpoint can be steered", ErrInvalid)
	}
	return r, nil
}

// SteerLong changes the sub-questions a long run hasn't asked yet, at a checkpoint.
func (l Legwork) SteerLong(ctx context.Context, s tenancy.Scope, id string, st Steer) (ResearchView, error) {
	r, err := l.atCheckpoint(ctx, s, id)
	if err != nil {
		return ResearchView{}, err
	}
	drop := map[int]bool{}
	for _, i := range st.Drop {
		if i < r.Long.Asked || i >= len(r.Steps) {
			return ResearchView{}, fmt.Errorf("%w: only a sub-question not yet asked can be dropped", ErrInvalid)
		}
		drop[i] = true
	}
	steps := append([]string{}, r.Steps[:r.Long.Asked]...)
	for i := r.Long.Asked; i < len(r.Steps); i++ {
		if !drop[i] {
			steps = append(steps, r.Steps[i])
		}
	}
	for _, a := range st.Add {
		a = strings.Join(strings.Fields(a), " ")
		if a == "" || utf8.RuneCountInString(a) > MaxStep || !printableText(a) {
			return ResearchView{}, fmt.Errorf("%w: a sub-question is 1 to %d characters", ErrInvalid, MaxStep)
		}
		steps = append(steps, a)
	}
	if len(steps) > MaxSteps {
		return ResearchView{}, fmt.Errorf("%w: a report has at most %d sub-questions", ErrInvalid, MaxSteps)
	}
	r.Steps = steps
	r.Long.EstimateUSD = float64(len(steps)) * StepEstimateUSD
	if err := l.audit(ctx, s.User(), r, "steered", hashOf(strings.Join(steps, "\n"))); err != nil {
		return ResearchView{}, err
	}
	return l.putLong(ctx, r)
}

// StopLong stops a long run at a checkpoint with what it has.
func (l Legwork) StopLong(ctx context.Context, s tenancy.Scope, id string) (ResearchView, error) {
	r, err := l.atCheckpoint(ctx, s, id)
	if err != nil {
		return ResearchView{}, err
	}
	r.State, r.Stopped, r.Long.Checkpoint = StateDone, "asker", nil
	if err := l.finish(ctx, r); err != nil {
		return ResearchView{}, err
	}
	return l.recheck(ctx, s, r)
}

// GoOn lets a long run at a checkpoint go on at the next notify run.
func (l Legwork) GoOn(ctx context.Context, s tenancy.Scope, id string) (ResearchView, error) {
	r, err := l.atCheckpoint(ctx, s, id)
	if err != nil {
		return ResearchView{}, err
	}
	r.Long.Checkpoint = nil
	if err := l.audit(ctx, s.User(), r, "go_on", hashOf(r.Question)); err != nil {
		return ResearchView{}, err
	}
	return l.putLong(ctx, r)
}

func (l Legwork) putLong(ctx context.Context, r Research) (ResearchView, error) {
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return ResearchView{}, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	return view(r, r.Sections, 0), nil
}

// runStage runs a long report's next stage as its owner now: the earlier sections checked
// again first, then up to PerStage sub-questions within the ceiling; then a checkpoint, or
// the end.
func (l Legwork) runStage(ctx context.Context, r Research) (Research, error) {
	now := l.now()
	if r.Long.Started.IsZero() {
		r.Long.Started = now
	}
	r.Long.Checkpoint = nil
	s, err := l.ScopeIn(ctx, r.User, r.Space)
	if err != nil {
		r.State, r.Why = StateFailed, "its owner is no longer active"
		return r, l.finish(ctx, r)
	}
	var kept []Section // LAW-26: what the owner lost since the last stage goes now
	for _, sec := range r.Sections {
		claims, d, err := l.Agent.Recheck(ctx, s, sec.Claims)
		if err != nil {
			return r, err
		}
		r.Long.Dropped += d
		if sec.Claims = claims; anyCited(claims) {
			kept = append(kept, sec)
		}
	}
	r.Sections = kept
	switch {
	case now.Sub(r.Long.Started) >= MaxLongRun:
		r.State, r.Stopped = StateDone, "time"
		return r, l.finish(ctx, r)
	case r.CostUSD >= r.Long.LimitUSD:
		r.State, r.Stopped = StateDone, "cap"
		return r, l.finish(ctx, r)
	}
	tctx, tally := budget.WithTally(ctx)
	skill, _ := SkillName(r.Question)
	end := min(len(r.Steps), r.Long.Asked+r.Long.PerStage)
	limited, start := false, r.Long.Asked
	for _, step := range r.Steps[r.Long.Asked:end] {
		if r.CostUSD+tally.USD() >= r.Long.LimitUSD {
			r.Stopped = "cap"
			break
		}
		asked := step
		if skill != "" {
			asked = "/" + skill + " " + step
		}
		resp, err := l.Agent.Ask(tctx, s, Request{Question: asked, Agent: AgentResearch, Documents: r.Documents})
		switch {
		case errors.Is(err, ErrRate): // its owner's asks this minute are spent: the rest of the stage waits
			limited = true
		case errors.Is(err, ErrBudget):
			r.Stopped = "budget"
		case errors.Is(err, ErrKilled):
			r.State, r.Why = StateFailed, "the research agent is turned off"
		case err != nil:
			r.State, r.Why = StateFailed, "a step couldn't be answered"
		}
		if err != nil {
			break
		}
		r.Long.Asked++
		if claims, _ := citedOnly(resp.Answer.Claims); !claims[0].Uncited {
			r.Sections = append(r.Sections, Section{Question: step, Trace: resp.Trace, Claims: claims, Assumed: resp.Assumed})
		}
	}
	r.CostUSD += tally.USD()
	if limited && r.Long.Asked == start && r.Stopped == "" && r.State != StateFailed {
		r.State = StateQueued // nothing of the stage was asked: it runs at the next notify run
		if err := l.Research.PutResearch(ctx, r); err != nil {
			return r, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
		}
		return r, nil
	}
	r.Long.Stage++
	if r.State == StateFailed || r.Stopped != "" || r.Long.Asked >= len(r.Steps) || r.Long.Stage >= MaxStages {
		if r.State != StateFailed {
			r.State = StateDone
		}
		return r, l.finish(ctx, r)
	}
	t := now
	r.State, r.Long.Checkpoint = StateQueued, &t
	if err := l.audit(ctx, r.User, r, "checkpoint", fmt.Sprintf("stage %d", r.Long.Stage)); err != nil {
		return r, err
	}
	if err := l.Research.PutResearch(ctx, r); err != nil {
		return r, fmt.Errorf("%w: reports: %v", ErrUnavailable, err)
	}
	return r, nil
}
