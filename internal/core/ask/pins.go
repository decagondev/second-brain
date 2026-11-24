package ask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/tenancy"
)

// Verified answers (sb-ask(3), Questions to the stewards; SB-3601): a steward pins a good
// reply as the space's answer to its question, and a close new ask in the space shows it
// first, every cite checked again as the asker.
const (
	MaxPins        = 50
	MaxPinQuestion = 300
	PinAt          = LikeAt // the cosine a pin needs
)

// Pin is one verified answer.
type Pin struct {
	ID       string    `json:"id"`
	Question string    `json:"question"`
	Text     string    `json:"text"`
	Cites    []string  `json:"cites"`
	From     string    `json:"from"` // the steward question it was pinned from
	By       string    `json:"by"`
	At       time.Time `json:"at"`
}

// Pins is one space's verified answers.
type Pins struct {
	Tenant string    `json:"tenant"`
	Space  string    `json:"space"` // spc_NAME
	Items  []Pin     `json:"items"`
	At     time.Time `json:"at"`
}

// PinStore keeps one record of pins per space.
type PinStore interface {
	PutPins(ctx context.Context, p Pins) error
	Pins(ctx context.Context, tenant, space string) (Pins, bool, error)
}

// Pinned is a pin as an asker is shown it.
type Pinned struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Text     string   `json:"text"`
	Cites    []string `json:"cites"`
	Verified string   `json:"verified"` // YYYY-MM-DD
}

// PinView is a pin on the steward screen.
type PinView struct {
	Pin
	Due bool `json:"due,omitempty"`
}

func (x Stewards) pins(ctx context.Context, name string) (Pins, error) {
	if x.Agent.Pins == nil {
		return Pins{}, fmt.Errorf("%w: this cell keeps no verified answers", ErrUnavailable)
	}
	p, _, err := x.Agent.Pins.Pins(ctx, x.Agent.Tenant, SpaceKey(name))
	if err != nil {
		return Pins{}, fmt.Errorf("%w: pins: %v", ErrUnavailable, err)
	}
	p.Tenant, p.Space = x.Agent.Tenant, SpaceKey(name)
	return p, nil
}

// PinsOf lists the space's pins for its stewards, newest first.
func (x Stewards) PinsOf(ctx context.Context, s tenancy.Scope, name string) ([]PinView, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return nil, err
	}
	out := []PinView{}
	if x.Agent.Pins == nil {
		return out, nil
	}
	p, err := x.pins(ctx, name)
	if err != nil {
		return nil, err
	}
	after := x.Agent.reviewAfter([]string{SpaceKey(name)})
	now := x.Now()
	for i := len(p.Items) - 1; i >= 0; i-- {
		it := p.Items[i]
		out = append(out, PinView{Pin: it, Due: pinDue(it, after, now)})
	}
	return out, nil
}

func pinDue(p Pin, after int, now time.Time) bool {
	return now.Sub(p.At) > time.Duration(after)*24*time.Hour
}

// PinReply pins the reply to question id of the space as its verified answer, with
// question as askers will see it ("" keeps the question as asked).
func (x Stewards) PinReply(ctx context.Context, s tenancy.Scope, name, id, question string) (Pin, error) {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return Pin{}, err
	}
	q, err := x.question(ctx, name, id)
	if errors.Is(err, ErrNotSteward) {
		return Pin{}, fmt.Errorf("%w: no such question in this space", ErrInvalid)
	}
	if err != nil {
		return Pin{}, err
	}
	if q.Reply == nil || len(q.Reply.Cites) == 0 {
		return Pin{}, fmt.Errorf("%w: only a reply with cites can be pinned", ErrInvalid)
	}
	question = strings.TrimSpace(question)
	if question == "" {
		question = strings.TrimSpace(q.Question)
	}
	if question == "" || utf8.RuneCountInString(question) > MaxPinQuestion || strings.ContainsAny(question, "\r\n") {
		return Pin{}, fmt.Errorf("%w: a pinned question is one line of 1 to %d characters", ErrInvalid, MaxPinQuestion)
	}
	p, err := x.pins(ctx, name)
	if err != nil {
		return Pin{}, err
	}
	pin := Pin{ID: x.Agent.NewID("pin_"), Question: question, Text: q.Reply.Text, Cites: append([]string(nil), q.Reply.Cites...), From: q.ID, By: s.User(), At: x.Now().UTC()}
	kept := p.Items[:0:0]
	for _, it := range p.Items {
		if it.From == q.ID {
			pin.ID = it.ID // pinning it again renews it
			continue
		}
		kept = append(kept, it)
	}
	if len(kept) >= MaxPins {
		return Pin{}, fmt.Errorf("%w: a space keeps at most %d verified answers", ErrInvalid, MaxPins)
	}
	sum := sha256.Sum256([]byte(pin.Question + "\n" + pin.Text))
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: "pin:" + name, State: "pinned",
		PayloadSHA256: hex.EncodeToString(sum[:]), Reason: pin.ID, Space: auditSpace(SpaceKey(name))}); err != nil {
		return Pin{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	kept = append(kept, pin)
	p.Items, p.At = kept, pin.At
	if err := x.Agent.Pins.PutPins(ctx, p); err != nil {
		return Pin{}, fmt.Errorf("%w: pins: %v", ErrUnavailable, err)
	}
	return pin, nil
}

// Unpin removes one of the space's pins.
func (x Stewards) Unpin(ctx context.Context, s tenancy.Scope, name, id string) error {
	if _, _, err := x.steward(ctx, s, name); err != nil {
		return err
	}
	p, err := x.pins(ctx, name)
	if err != nil {
		return err
	}
	kept := p.Items[:0:0]
	for _, it := range p.Items {
		if it.ID != id {
			kept = append(kept, it)
		}
	}
	if len(kept) == len(p.Items) {
		return fmt.Errorf("%w: no such verified answer in this space", ErrInvalid)
	}
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "config", Subject: "pin:" + name, State: "unpinned",
		Reason: id, Space: auditSpace(SpaceKey(name))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	p.Items, p.At = kept, x.Now().UTC()
	if err := x.Agent.Pins.PutPins(ctx, p); err != nil {
		return fmt.Errorf("%w: pins: %v", ErrUnavailable, err)
	}
	return nil
}

// pinned is the space's pin closest to q at PinAt or more, younger than the space's
// review_after_days, shown only if the asker can open every one of its cites now. Anything
// that can't be read shows no pin: a pin is extra, never needed for the answer.
func (a Agent) pinned(ctx context.Context, s tenancy.Scope, q string) *Pinned {
	if a.Pins == nil || a.Embedder == nil || s.Space() == "" {
		return nil
	}
	p, ok, err := a.Pins.Pins(ctx, a.Tenant, s.Space())
	if err != nil || !ok || len(p.Items) == 0 {
		return nil
	}
	after, now := a.reviewAfter([]string{s.Space()}), a.Now()
	var live []Pin
	texts := []string{q}
	for _, it := range p.Items {
		if !pinDue(it, after, now) && len(it.Cites) > 0 {
			live = append(live, it)
			texts = append(texts, it.Question)
		}
	}
	if len(live) == 0 {
		return nil
	}
	vs, err := a.Embedder.Embed(ctx, texts)
	if err != nil || len(vs) != len(texts) {
		return nil
	}
	best, at := -1, PinAt
	for i := range live {
		if c := cosine(vs[0], vs[i+1]); c >= at {
			best, at = i, c
		}
	}
	if best < 0 {
		return nil
	}
	pin := live[best]
	ok2, _, err := a.CheckCites(ctx, s, pin.Cites)
	if err != nil {
		return nil
	}
	open := map[string]bool{}
	for _, r := range ok2 {
		open[r.Cite] = true
	}
	for _, c := range pin.Cites {
		if !open[c] {
			return nil // one cite they can't open hides the whole pin (LAW-01)
		}
	}
	return &Pinned{ID: pin.ID, Question: pin.Question, Text: pin.Text, Cites: pin.Cites, Verified: pin.At.UTC().Format(time.DateOnly)}
}

// PinnedLead is the chat apps' words before a pinned answer (SB-3601).
func PinnedLead(p Pinned) string {
	return "Verified by the stewards on " + p.Verified + ":"
}
