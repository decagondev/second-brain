package ask

import (
	"regexp"
	"strings"
)

// The post-incident review (sb-agents(7), Scheduled and triggered agents; SB-4604): when an
// incident that had a brief closes, a draft of what happened, what was decided and changed
// while it was open, and its follow-ups, for the steward it ran as to publish or discard.
const (
	AgentReview = "incident-review"
	DraftReview = "review"
)

var statusLineRe = regexp.MustCompile(`(?i)^\s*status\s*:\s*(.+)$`)

// closedWords start a closed ticket's status name.
var closedWords = []string{"done", "closed", "resolved", "completed", "cancel"}

// IsClosed says a ticket's text has a "Status:" line among its first five lines that says
// it is closed: "(closed)" after the name, or a name that starts done, closed, resolved,
// completed or cancel (any case).
func IsClosed(text string) bool {
	lines := strings.SplitN(text, "\n", 6)
	if len(lines) > 5 {
		lines = lines[:5]
	}
	for _, l := range lines {
		m := statusLineRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		v := strings.ToLower(strings.TrimSpace(m[1]))
		if strings.Contains(v, "(closed)") {
			return true
		}
		for _, w := range closedWords {
			if strings.HasPrefix(v, w) {
				return true
			}
		}
		return false
	}
	return false
}

// ReviewQuestion is a post-incident review's question: the ticket's title is a topic only.
func ReviewQuestion(title string) string {
	return "For the incident " + cut(title, 200) + ", now closed: what happened when, what was decided and changed while it was open, and what follow-ups remain?"
}
