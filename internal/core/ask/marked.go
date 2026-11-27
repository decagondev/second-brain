package ask

import "strings"

// Marked as AI (sb-agents(7), Marked as AI; SB-4706, LAW-28): anything an agent wrote says
// which agent and version wrote it wherever it is shown, until a person rewrites it.

// Mark is who wrote a text: an agent, and its version, the release it ran under.
type Mark struct {
	Agent   string `json:"agent"`
	Version string `json:"version,omitempty"`
}

// MarkLine is the mark as it is shown: "Written by case-brief, version rel_7".
func MarkLine(agent, version string) string {
	if agent == "" {
		return ""
	}
	if version == "" {
		return "Written by " + agent
	}
	return "Written by " + agent + ", version " + version
}

// ReadMark is the mark of a passage a model read (SB-6101, SB-6102, LAW-28); "" for the
// source's own text.
func ReadMark(readBy string) string {
	switch readBy {
	case "":
		return ""
	case "vision":
		return "Read by a model from a scan or picture"
	case "transcribe":
		return "Transcribed by a model from a recording"
	}
	return "Read by a model"
}

// GeneratedBy is the front matter's AGENT@VERSION, or AGENT alone without a version.
func GeneratedBy(agent, version string) string {
	if version == "" {
		return agent
	}
	return agent + "@" + version
}

// draftMark is an agent draft's mark; nil for a thread a person saved.
func draftMark(t *SavedThread) *Mark {
	if t == nil || t.Agent == "" {
		return nil
	}
	return &Mark{Agent: t.Agent, Version: t.Release}
}

// Rewrote is a reply in which none of the lines of an agent's draft (its text now or any
// earlier version) are left: the person wrote it, so it carries no mark.
func Rewrote(t *SavedThread, reply string) bool {
	if t == nil || t.Agent == "" {
		return true
	}
	have := map[string]bool{}
	for _, l := range strings.Split(reply, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			have[l] = true
		}
	}
	for _, txt := range append([]string{t.Text}, t.Versions...) {
		for _, l := range strings.Split(txt, "\n") {
			if l = strings.TrimSpace(l); l != "" && have[l] {
				return false
			}
		}
	}
	return true
}

// markFront is the front matter of a note published from an agent draft: generated_by and
// approved_by, and edited when the reply isn't the draft's text. None for a rewrite.
func markFront(q StewardQuestion, approver string, people ...string) []string {
	if q.Reply == nil || Rewrote(q.Thread, q.Reply.Text) {
		return nil
	}
	out := []string{"generated_by: " + GeneratedBy(q.Thread.Agent, q.Thread.Release), "approved_by: " + strings.ToLower(approver)}
	if len(people) > 0 && people[0] != "" { // SB-5204: who asked for it
		out = append(out, "asked_by: "+strings.ToLower(people[0]))
	}
	if len(people) > 1 && people[1] != "" { // and who edited it in place
		out = append(out, "edited_by: "+strings.ToLower(people[1]))
	}
	gen := q.Thread.Text
	if q.Thread.Generated != "" { // edited in place (SB-5202)
		gen = q.Thread.Generated
	}
	if strings.TrimSpace(q.Reply.Text) != strings.TrimSpace(gen) {
		out = append(out, "edited: true")
	}
	return out
}

// CiteMark is a cited note's mark as shown (SB-4706): "Written by case-brief, version
// rel_7, approved by sam@…", with "edited and approved" when the person changed it; "" for
// a document a person wrote.
func CiteMark(generatedBy, approvedBy string, edited bool, people ...string) string {
	if generatedBy == "" {
		return ""
	}
	agent, version, _ := strings.Cut(generatedBy, "@")
	line := MarkLine(agent, version)
	if len(people) > 0 && people[0] != "" { // SB-5204
		line += ", asked for by " + people[0]
	}
	if len(people) > 1 && people[1] != "" {
		line += ", edited by " + people[1]
	}
	if approvedBy != "" {
		if edited {
			line += ", edited and approved by " + approvedBy
		} else {
			line += ", approved by " + approvedBy
		}
	}
	return line
}
