package laws

import (
	"fmt"
	"strings"
)

// LAW-01 through the chat app (SB-308): an answer asked from Slack cites only documents
// the asker can open now, the source was asked about the asker (not someone else), and
// the reply is visible only to the asker.

// ChatReply is what the chat app sent back for one /ask.
type ChatReply struct {
	Docs      []string // documents cited in the reply
	VisibleTo string   // "ephemeral:<slack user>" or "channel"
}

// ChatAsk sends /ask as the world user's chat account and returns the final reply.
type ChatAsk func(w *World, userID, question string) (ChatReply, error)

// NewChatAsk builds the system under test.
var NewChatAsk = func() ChatAsk {
	return func(*World, string, string) (ChatReply, error) {
		return ChatReply{}, fmt.Errorf("%w: the chat app arrives with SB-308", ErrPending)
	}
}

// ChatUser is the chat account id of a world user (its email is id@tenant.example).
func ChatUser(userID string) string { return "U" + strings.ToUpper(userID) }

func law01Chat(w *World, ask ChatAsk) error {
	for _, u := range w.Users {
		rep, err := ask(w, u.ID, "anything")
		if err != nil {
			return err
		}
		if rep.VisibleTo != "ephemeral:"+ChatUser(u.ID) {
			return fmt.Errorf("%s's answer was visible to %q, not only to them", u.ID, rep.VisibleTo)
		}
		for _, d := range rep.Docs {
			if !w.truthAllow(u.ID, d) {
				return fmt.Errorf("%s was cited %s in chat, which the source does not allow now", u.ID, d)
			}
			if !w.asked[u.ID+"|"+d] {
				return fmt.Errorf("%s was cited %s in chat without the source being asked about them", u.ID, d)
			}
		}
	}
	return nil
}
