package msgraph

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

// Shared mailboxes (sb-agents(7), Start on a new email; SB-5907): the one shared mailbox a
// space names, read with a delta query from a stored link, so each new message is seen
// once. Only a mailbox the manifest names is ever asked for, and Graph's application access
// policy limits Mail.Read to those (ADR-0023); a person's own mailbox is never read. A
// message's text and attachments are data for the space's agent, never instructions.

// Mail is one new message.
type Mail struct {
	ID, From, Subject, Body string
	Attachments             []Attachment
}

// Attachment is one file attached to a message.
type Attachment struct {
	Name string
	Data []byte
}

// Mailbox reads shared mailboxes.
type Mailbox struct {
	Graph *Client
	// MaxAttachment is the largest attachment read; larger ones are named only. 0 is 2 MB.
	MaxAttachment int
}

// New lists the messages of box since link ("" starts from now: the messages already there
// are left alone) and the link for next time.
func (m *Mailbox) New(ctx context.Context, box, link string) ([]Mail, string, error) {
	start := link == ""
	if start {
		link = "/users/" + url.PathEscape(strings.ToLower(box)) + "/mailFolders/inbox/messages/delta?$select=id,subject,body,from,hasAttachments"
	}
	msgs, delta, _, err := all[struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		Body    struct {
			Content string `json:"content"`
		} `json:"body"`
		From struct {
			Email struct {
				Address string `json:"address"`
			} `json:"emailAddress"`
		} `json:"from"`
		HasAttachments bool `json:"hasAttachments"`
	}](ctx, m.Graph, link, 0)
	if err != nil {
		return nil, link, err
	}
	if start {
		return nil, delta, nil
	}
	var out []Mail
	for _, x := range msgs {
		mail := Mail{ID: x.ID, From: strings.ToLower(x.From.Email.Address), Subject: x.Subject, Body: x.Body.Content}
		if x.HasAttachments {
			as, err := m.attachments(ctx, box, x.ID)
			if err != nil {
				return nil, link, err
			}
			mail.Attachments = as
		}
		out = append(out, mail)
	}
	return out, delta, nil
}

func (m *Mailbox) attachments(ctx context.Context, box, id string) ([]Attachment, error) {
	max := m.MaxAttachment
	if max <= 0 {
		max = 2 << 20
	}
	as, _, _, err := all[struct {
		Type    string          `json:"@odata.type"`
		Name    string          `json:"name"`
		Content json.RawMessage `json:"contentBytes"`
	}](ctx, m.Graph, "/users/"+url.PathEscape(strings.ToLower(box))+"/messages/"+url.PathEscape(id)+"/attachments", 20)
	if err != nil {
		return nil, err
	}
	var out []Attachment
	for _, a := range as {
		if a.Type != "#microsoft.graph.fileAttachment" {
			continue
		}
		var b64 string
		_ = json.Unmarshal(a.Content, &b64)
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil || len(data) > max {
			data = nil // too large or unreadable: named only
		}
		out = append(out, Attachment{Name: a.Name, Data: data})
	}
	return out, nil
}
