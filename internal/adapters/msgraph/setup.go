package msgraph

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
)

// The checks sb selfhost check-microsoft runs (SB-6003): what the app registration was
// granted, whether each named drive answers, which sensitivity labels the tenant defines,
// and whether a mailbox can be read. They read only.

// Granted are the application permissions the cell's app registration holds, by value
// (Files.Read.All), sorted.
func (c *Client) Granted(ctx context.Context) ([]string, error) {
	raw, err := c.Creds(ctx)
	if err != nil {
		return nil, err
	}
	var cr Credentials
	if json.Unmarshal([]byte(raw), &cr) != nil || cr.ClientID == "" {
		return nil, errors.New("msgraph: the credentials name no client_id")
	}
	var sp struct {
		ID string `json:"id"`
	}
	if err := c.get(ctx, "/servicePrincipals(appId='"+url.PathEscape(cr.ClientID)+"')", &sp); err != nil {
		return nil, err
	}
	as, _, _, err := all[struct {
		AppRoleID  string `json:"appRoleId"`
		ResourceID string `json:"resourceId"`
	}](ctx, c, "/servicePrincipals/"+url.PathEscape(sp.ID)+"/appRoleAssignments", 0)
	if err != nil {
		return nil, err
	}
	roles := map[string]map[string]string{} // resource -> role id -> value
	var out []string
	for _, a := range as {
		if roles[a.ResourceID] == nil {
			var r struct {
				AppRoles []struct{ ID, Value string } `json:"appRoles"`
			}
			if err := c.get(ctx, "/servicePrincipals/"+url.PathEscape(a.ResourceID)+"?$select=appRoles", &r); err != nil {
				return nil, err
			}
			roles[a.ResourceID] = map[string]string{}
			for _, x := range r.AppRoles {
				roles[a.ResourceID][x.ID] = x.Value
			}
		}
		v := roles[a.ResourceID][a.AppRoleID]
		if v == "" {
			v = "unknown:" + a.AppRoleID // a role Graph doesn't name is not read-only
		}
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

// readOnly are the application permissions the cell may hold (ADR-0023).
var readOnly = map[string]bool{
	"Sites.Read.All": true, "Sites.Selected": true, "Files.Read.All": true, "Group.Read.All": true, "GroupMember.Read.All": true,
	"User.Read.All": true, "Calendars.Read": true, "OnlineMeetingTranscript.Read.All": true, "OnlineMeetings.Read.All": true,
	"Mail.Read": true, "InformationProtectionPolicy.Read.All": true, "Directory.Read.All": true, "Application.Read.All": true,
}

// ReadOnly says whether an application permission only reads. Anything not on the list,
// including a role Graph didn't name, doesn't (LAW-03).
func ReadOnly(perm string) bool { return readOnly[perm] }

// DriveAnswers reads a drive's root.
func (c *Client) DriveAnswers(ctx context.Context, drive string) error {
	_, err := c.raw(ctx, "/drives/"+url.PathEscape(drive)+"/root")
	return err
}

// LabelIDs are the sensitivity labels the tenant defines.
func (c *Client) LabelIDs(ctx context.Context) ([]string, error) {
	ls, _, _, err := all[struct {
		ID string `json:"id"`
	}](ctx, c, "/security/informationProtection/sensitivityLabels", 0)
	var out []string
	for _, l := range ls {
		out = append(out, l.ID)
	}
	sort.Strings(out)
	return out, err
}

// CanRead says whether the app can read mailbox's inbox: false for a refusal (401, 403),
// an error for anything else.
func (c *Client) CanRead(ctx context.Context, mailbox string) (bool, error) {
	_, err := c.raw(ctx, "/users/"+url.PathEscape(mailbox)+"/mailFolders/inbox/messages?$top=1&$select=id")
	var ge *Error
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &ge) && (ge.Status == 401 || ge.Status == 403):
		return false, nil
	}
	return false, err
}
