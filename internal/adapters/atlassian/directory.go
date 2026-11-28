package atlassian

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Directory is the site's Atlassian users and groups (SB-6202): who a person is by email,
// which groups they are in, and a group's members, read live. Anything the site can't say
// is an error, never an empty answer (LAW-03).
type Directory struct{ Client *Client }

// MaxMembers is the most members of one group read; a larger group is ErrTooLarge.
const MaxMembers = 5000

// ErrTooLarge is a group too large to expand, counted and never guessed.
var ErrTooLarge = errors.New("atlassian: the group is too large to expand")

// ErrNoUser is an email no active Atlassian account has.
var ErrNoUser = errors.New("atlassian: no active account has that email")

type user struct {
	AccountID   string `json:"accountId"`
	Email       string `json:"emailAddress"`
	Active      bool   `json:"active"`
	AccountType string `json:"accountType"`
}

// AccountID is the active account with email.
func (d *Directory) AccountID(ctx context.Context, email string) (string, error) {
	var us []user
	if err := d.Client.get(ctx, "/rest/api/3/user/search?query="+url.QueryEscape(email), &us); err != nil {
		return "", err
	}
	for _, u := range us {
		if strings.EqualFold(u.Email, email) && u.Active && u.AccountType == "atlassian" {
			return u.AccountID, nil
		}
	}
	return "", ErrNoUser
}

// GroupsOf are the names of the groups email's account is in, sorted.
func (d *Directory) GroupsOf(ctx context.Context, email string) ([]string, error) {
	id, err := d.AccountID(ctx, email)
	if err != nil {
		return nil, err
	}
	var gs []struct {
		Name string `json:"name"`
	}
	if err := d.Client.get(ctx, "/rest/api/3/user/groups?accountId="+url.QueryEscape(id), &gs); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(gs))
	for _, g := range gs {
		out = append(out, strings.ToLower(g.Name))
	}
	sort.Strings(out)
	return out, nil
}

// Members are the emails of a group's active members, sorted; ErrTooLarge past MaxMembers.
func (d *Directory) Members(ctx context.Context, group string) ([]string, error) {
	var out []string
	for start := 0; ; {
		var p struct {
			Values []user `json:"values"`
			IsLast bool   `json:"isLast"`
		}
		if err := d.Client.get(ctx, "/rest/api/3/group/member?groupname="+url.QueryEscape(group)+"&maxResults=50&startAt="+strconv.Itoa(start), &p); err != nil {
			return nil, err
		}
		for _, u := range p.Values {
			if u.Active && u.Email != "" {
				out = append(out, strings.ToLower(u.Email))
			}
		}
		start += len(p.Values)
		if len(out) > MaxMembers {
			return nil, ErrTooLarge
		}
		if p.IsLast || len(p.Values) == 0 {
			break
		}
	}
	sort.Strings(out)
	return out, nil
}
