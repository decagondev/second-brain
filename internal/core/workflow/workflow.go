// Package workflow holds the gated workflow templates (sb-approvals(7), The ticket-update
// workflow; SB-410) and turns a manifest's workflows into the tool bus's registry.
package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"example.com/second-brain/internal/core/approvals"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/toolbus"
)

// MaxComment is the longest ticket-update comment, in characters.
const MaxComment = 10000

// Template is one workflow's fixed shape.
type Template struct {
	Tool      string
	Connector string
	Schema    string // the payload's JSON Schema, pinned by the release bundle (SB-503)
	Validate  func(payload []byte) error
}

// Templates are the known workflows; manifest.Workflows names the same set.
var Templates = map[string]Template{
	"ticket-update": {Tool: "ticket-update.comment", Connector: "tickets", Validate: validateComment,
		Schema: `{"type":"object","additionalProperties":false,"required":["comment"],"properties":{"comment":{"type":"string","minLength":1,"maxLength":10000}}}`},
}

func validateComment(payload []byte) error {
	d := json.NewDecoder(bytes.NewReader(payload))
	d.DisallowUnknownFields()
	var p struct {
		Comment *string `json:"comment"`
	}
	if err := d.Decode(&p); err != nil || d.More() {
		return fmt.Errorf("%w: ticket-update takes {\"comment\": \"…\"} and nothing else", approvals.ErrPayload)
	}
	if p.Comment == nil || strings.TrimSpace(*p.Comment) == "" || utf8.RuneCountInString(*p.Comment) > MaxComment {
		return fmt.Errorf("%w: comment must be 1 to %d characters", approvals.ErrPayload, MaxComment)
	}
	return nil
}

// ErrUnknown is a manifest workflow with no template.
var ErrUnknown = errors.New("workflow: unknown workflow")

// Registry is the tool registry for a tenant's manifest: one consequential tool per
// enabled workflow, approved by the manifest's approval group.
func Registry(t manifest.Tenant) (map[string]toolbus.Tool, error) {
	out := map[string]toolbus.Tool{}
	for _, w := range t.Workflows {
		tpl, ok := Templates[w]
		if !ok {
			return nil, fmt.Errorf("%w %q", ErrUnknown, w)
		}
		group, ok := strings.CutPrefix(t.Approval[w], "group:")
		if !ok || group == "" {
			return nil, fmt.Errorf("workflow: %q has no approval group", w)
		}
		out[tpl.Tool] = toolbus.Tool{Name: tpl.Tool, Connector: tpl.Connector, Consequential: true, Workflow: w, Approvers: group, Validate: tpl.Validate}
	}
	return out, nil
}
