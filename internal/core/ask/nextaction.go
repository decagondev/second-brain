package ask

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/tenancy"
)

// Draft the next action (sb-ask(3), Draft the next action; SB-5304): an answer, brief or
// review becomes a draft for the person to edit: a comment on a Linear ticket, which goes
// through the gated ticket-update workflow like any action, or a message the bot posts in
// one of the space's channels, marked as sent by the person. Nothing goes out until the
// person sends it.

// Kinds of a drafted action.
const (
	ActionTicket  = "ticket"
	ActionMessage = "message"
	// ActionJiraIssue is a new Jira issue the person files themselves (SB-6207).
	ActionJiraIssue = "jira-issue"

	MaxDraftedAction = 4000
)

// ActionDraft is a drafted next action: its kind, the text to edit, and the cites it names,
// each one the person can open now.
type ActionDraft struct {
	Kind  string   `json:"kind"`
	Text  string   `json:"text"`
	Cites []string `json:"cites"`
	// For a Jira issue (SB-6207): its summary, and the link that opens Jira's own create form
	// filled in, which the person files under their own sign-in; Warning when it cites a
	// confidential or restricted document the project's readers may not open (LAW-32).
	Summary string `json:"summary,omitempty"`
	Link    string `json:"link,omitempty"`
	Warning string `json:"warning,omitempty"`
}

// JiraWarning is the warning on a Jira issue drafted from labelled sources.
const JiraWarning = "This draft cites confidential or restricted documents. People who can see the Jira project may not be allowed to open them: check before you file it."

// DraftAction drafts the next action from s's own answer trace, its cites checked again as
// s: a cite they can no longer open is left out, and so is its claim.
func (a Agent) DraftAction(ctx context.Context, s tenancy.Scope, trace, kind string) (ActionDraft, error) {
	if kind != ActionTicket && kind != ActionMessage && kind != ActionJiraIssue {
		return ActionDraft{}, fmt.Errorf("%w: kind is ticket, message or jira-issue", ErrInvalid)
	}
	if kind == ActionJiraIssue && a.JiraCreate == nil {
		return ActionDraft{}, fmt.Errorf("%w: this cell has no Jira", ErrUnavailable)
	}
	t, err := a.Trace(ctx, s, trace)
	if err != nil {
		return ActionDraft{}, err
	}
	claims, _, err := a.Recheck(ctx, s, t.Response.Answer.Claims)
	if err != nil {
		return ActionDraft{}, err
	}
	out := ActionDraft{Kind: kind, Cites: []string{}}
	var lines []string
	seen := map[string]bool{}
	labelled := false
	for _, c := range claims {
		if c.Uncited || len(c.Cites) == 0 {
			continue
		}
		lines = append(lines, "- "+strings.TrimSpace(c.Text))
		for _, ci := range c.Cites {
			labelled = labelled || ci.Label == "confidential" || ci.Label == "restricted"
			if !seen[ci.Cite] {
				seen[ci.Cite] = true
				out.Cites = append(out.Cites, ci.Cite)
			}
		}
	}
	if len(lines) == 0 {
		return ActionDraft{}, fmt.Errorf("%w: nothing in this answer you can still open to draft from", ErrInvalid)
	}
	text := strings.Join(lines, "\n")
	if len(out.Cites) > 0 {
		var srcs []string
		for _, c := range out.Cites {
			srcs = append(srcs, "- "+c)
		}
		text += "\n\nSources:\n" + strings.Join(srcs, "\n")
	}
	if r := []rune(text); len(r) > MaxDraftedAction {
		text = string(r[:MaxDraftedAction])
	}
	out.Text = text
	if kind == ActionJiraIssue { // SB-6207: filed by the person in Jira, never by the brain
		out.Summary = cut(strings.TrimPrefix(lines[0], "- "), 200)
		out.Link = a.JiraCreate(out.Summary, text)
		if labelled {
			out.Warning = JiraWarning
		}
	}
	return out, nil
}

// ChannelPoster posts one message in a chat channel as the bot, returning its id.
type ChannelPoster interface {
	Post(ctx context.Context, channel, text string) (string, error)
}

// SentMessage is a message the bot posted for a person.
type SentMessage struct {
	Channel string `json:"channel"`
	ID      string `json:"id"`
}

// SendMessage posts s's edited text in channel, one of space name's channels, marked as sent
// by them, after writing its audit row (LAW-04). Only a member of the space sends there.
func (x Stewards) SendMessage(ctx context.Context, s tenancy.Scope, name, channel, text string, post ChannelPoster, channels func(space string) []string) (SentMessage, error) {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > MaxDraftedAction {
		return SentMessage{}, fmt.Errorf("%w: a message is 1 to %d characters", ErrInvalid, MaxDraftedAction)
	}
	if post == nil || channels == nil {
		return SentMessage{}, fmt.Errorf("%w: this cell has no chat bot to post with", ErrUnavailable)
	}
	if err := x.member(ctx, s, name); err != nil {
		return SentMessage{}, err
	}
	if !contains(channels(name), channel) {
		return SentMessage{}, fmt.Errorf("%w: post in one of the space's channels", ErrInvalid)
	}
	who := x.emailOf(ctx, s.User())
	if who == "" {
		return SentMessage{}, fmt.Errorf("%w: you have no address to sign the message with", ErrUnavailable)
	}
	body, _ := ingest.MaskOut("Sent by " + who + " via Second Brain:\n" + text) // SB-5102: nothing sensitive goes out
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: "message:" + strings.TrimPrefix(channel, "#"), State: "message_sent",
		PayloadSHA256: hashOf(body), Reason: "a drafted next action, sent by its person", Space: auditSpace(SpaceKey(name))}); err != nil {
		return SentMessage{}, fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	id, err := post.Post(ctx, channel, body)
	if err != nil {
		return SentMessage{}, fmt.Errorf("%w: chat: %v", ErrUnavailable, err)
	}
	return SentMessage{Channel: channel, ID: id}, nil
}
