package msgraph

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
)

// Exchange is one step of the recording script and its raw answer (SB-6002).
type Exchange struct {
	Step   string
	Status int
	Body   []byte
}

// Script is the fixed set of reads sb record runs against a real tenant (SB-6002): each
// drive's delta, the first item's metadata and permissions, a person's groups and manager,
// and a shared mailbox's delta. It reads only, and returns each answer as it came.
func (c *Client) Script(ctx context.Context, drives []string, person, mailbox string) ([]Exchange, error) {
	var out []Exchange
	get := func(step, path string) []byte {
		b, err := c.rawWith(ctx, path, sharingChanges)
		ex := Exchange{Step: step, Status: 200, Body: b}
		var ge *Error
		if errors.As(err, &ge) {
			ex.Status, ex.Body = ge.Status, nil
		} else if err != nil {
			ex.Status, ex.Body = 0, nil
		}
		out = append(out, ex)
		return ex.Body
	}
	for _, d := range drives {
		b := get("drive-delta", "/drives/"+url.PathEscape(d)+"/root/delta")
		var p struct {
			Value []struct {
				ID   string          `json:"id"`
				File json.RawMessage `json:"file"`
			} `json:"value"`
		}
		_ = json.Unmarshal(b, &p)
		for _, it := range p.Value {
			if it.File == nil {
				continue
			}
			get("drive-item", "/drives/"+url.PathEscape(d)+"/items/"+url.PathEscape(it.ID))
			get("item-permissions", "/drives/"+url.PathEscape(d)+"/items/"+url.PathEscape(it.ID)+"/permissions")
			break
		}
	}
	if person != "" {
		get("user-groups", "/users/"+url.PathEscape(person)+"/transitiveMemberOf")
		get("user-manager", "/users/"+url.PathEscape(person)+"/manager")
	}
	if mailbox != "" {
		get("mailbox-delta", "/users/"+url.PathEscape(mailbox)+"/mailFolders/inbox/messages/delta")
	}
	if len(out) == 0 {
		return nil, errors.New("msgraph: nothing to record: name a drive, a person or a mailbox")
	}
	return out, nil
}
