package atlassian

import (
	"context"
	"net/url"
	"sort"
	"strings"
)

// The checks sb selfhost check-atlassian runs (SB-6203). An Atlassian API token carries its
// account's permissions, so read-only means a read-only account: the check asks Jira which
// write permissions the account holds, and refuses any.

// WritePermissions are the Jira permissions a read-only account must not hold.
var WritePermissions = []string{"CREATE_ISSUES", "EDIT_ISSUES", "DELETE_ISSUES", "ADD_COMMENTS", "TRANSITION_ISSUES", "ASSIGN_ISSUES",
	"ADMINISTER_PROJECTS", "ADMINISTER"}

// Myself is the account's email.
func (c *Client) Myself(ctx context.Context) (string, error) {
	var u user
	if err := c.get(ctx, "/rest/api/3/myself", &u); err != nil {
		return "", err
	}
	return u.Email, nil
}

// Writes are the write permissions the account holds, in project if not "", sorted.
func (c *Client) Writes(ctx context.Context, project string) ([]string, error) {
	q := "/rest/api/3/mypermissions?permissions=" + strings.Join(WritePermissions, ",")
	if project != "" {
		q += "&projectKey=" + url.QueryEscape(project)
	}
	var p struct {
		Permissions map[string]struct {
			HavePermission bool `json:"havePermission"`
		} `json:"permissions"`
	}
	if err := c.get(ctx, q, &p); err != nil {
		return nil, err
	}
	var out []string
	for k, v := range p.Permissions {
		if v.HavePermission {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

// ProjectAnswers reads a Jira project.
func (c *Client) ProjectAnswers(ctx context.Context, key string) error {
	return c.get(ctx, "/rest/api/3/project/"+url.PathEscape(key), &struct{}{})
}

// SpaceAnswers reads a Confluence space by key.
func (c *Client) SpaceAnswers(ctx context.Context, key string) error {
	_, err := (&Confluence{Client: c}).space(ctx, key)
	return err
}
