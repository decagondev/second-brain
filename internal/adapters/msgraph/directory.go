package msgraph

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Entra groups and people (sb-identity(7), Entra ID; SB-5903): a person's groups with
// nested membership, a group's members expanded through nested groups, and a person's
// manager, read from Entra through Graph as they are now. A group larger than MaxExpand is
// counted as too large, never guessed at. People and their leaving still come through
// Entra's SCIM provisioning into the cell's own endpoint (SB-107), so a leaver is revoked
// within 5 minutes (LAW-12).

// MaxExpand is the most members a group is expanded to.
const MaxExpand = 5000

// ErrTooLarge is a group with more members than MaxExpand.
var ErrTooLarge = errors.New("msgraph: the group is too large to expand")

// Directory is one tenant's Entra ID.
type Directory struct {
	Graph     *Client
	MaxExpand int
}

type dirObject struct {
	Type string `json:"@odata.type"`
	ID   string `json:"id"`
	Mail string `json:"mail"`
	UPN  string `json:"userPrincipalName"`
}

// GroupsOf are email's groups, nested ones included, as group:MAIL (or group:ID for a group
// without a mail), the way ACL copies name them.
func (d *Directory) GroupsOf(ctx context.Context, email string) ([]string, error) {
	if email == "" {
		return nil, nil
	}
	os, _, _, err := all[dirObject](ctx, d.Graph, "/users/"+url.PathEscape(strings.ToLower(email))+"/transitiveMemberOf?$select=id,mail", 0)
	if notFound(err) {
		return nil, nil // not in Entra: no groups, never someone else's
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, o := range os {
		if o.Type != "" && o.Type != "#microsoft.graph.group" {
			continue
		}
		if o.Mail != "" {
			out = append(out, "group:"+strings.ToLower(o.Mail))
		}
		if o.ID != "" {
			out = append(out, "group:"+strings.ToLower(o.ID))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Members are a group's members' emails, nested groups' members included; ErrTooLarge with
// the count reached for a group past MaxExpand.
func (d *Directory) Members(ctx context.Context, group string) ([]string, error) {
	max := d.MaxExpand
	if max <= 0 {
		max = MaxExpand
	}
	os, _, more, err := all[dirObject](ctx, d.Graph, "/groups/"+url.PathEscape(group)+"/transitiveMembers?$select=id,mail,userPrincipalName", max)
	if err != nil {
		return nil, err
	}
	if more {
		return nil, fmt.Errorf("%w: more than %d", ErrTooLarge, max)
	}
	var out []string
	for _, o := range os {
		if o.Type != "" && o.Type != "#microsoft.graph.user" {
			continue
		}
		e := o.Mail
		if e == "" {
			e = o.UPN
		}
		if e != "" {
			out = append(out, strings.ToLower(e))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Manager is email's manager's email, "" for none.
func (d *Directory) Manager(ctx context.Context, email string) (string, error) {
	var o dirObject
	err := d.Graph.get(ctx, "/users/"+url.PathEscape(strings.ToLower(email))+"/manager?$select=mail,userPrincipalName", &o)
	if notFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if o.Mail != "" {
		return strings.ToLower(o.Mail), nil
	}
	return strings.ToLower(o.UPN), nil
}

// Email is an Entra user id's email (a Teams activity names people by their object id).
func (d *Directory) Email(ctx context.Context, id string) (string, error) {
	var o dirObject
	if err := d.Graph.get(ctx, "/users/"+url.PathEscape(id)+"?$select=id,mail,userPrincipalName", &o); err != nil {
		return "", err
	}
	if o.Mail != "" {
		return strings.ToLower(o.Mail), nil
	}
	return strings.ToLower(o.UPN), nil
}

// Calendar reads a person's own Outlook calendar through Graph (sb-connector(3), Calendars;
// SB-5905), as gcal reads Google's: only the events of the person asked for, by their own
// address, never anyone else's. Nothing it reads is indexed or kept.
type Calendar struct {
	Graph *Client
}

var _ ports.Calendar = (*Calendar)(nil)

// MaxEvents is the most events one call returns.
const MaxEvents = 50

// Events implements ports.Calendar: email's own events starting in [from, to), declined and
// cancelled ones left out. A person Graph doesn't know reads no events.
func (c *Calendar) Events(ctx context.Context, _ string, email string, from, to time.Time) ([]ports.CalendarEvent, error) {
	if email == "" {
		return nil, nil
	}
	q := url.Values{"startDateTime": {from.UTC().Format(time.RFC3339)}, "endDateTime": {to.UTC().Format(time.RFC3339)},
		"$select": {"id,subject,bodyPreview,start,attendees,isCancelled,responseStatus"}, "$top": {fmt.Sprint(MaxEvents)}}
	evs, _, _, err := all[struct {
		ID        string `json:"id"`
		Subject   string `json:"subject"`
		Body      string `json:"bodyPreview"`
		Cancelled bool   `json:"isCancelled"`
		Response  struct {
			Response string `json:"response"`
		} `json:"responseStatus"`
		Start struct {
			DateTime string `json:"dateTime"`
		} `json:"start"`
		Attendees []struct {
			Email struct {
				Address string `json:"address"`
			} `json:"emailAddress"`
		} `json:"attendees"`
	}](ctx, c.Graph, "/users/"+url.PathEscape(strings.ToLower(email))+"/calendarView?"+q.Encode(), MaxEvents)
	if notFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []ports.CalendarEvent
	for _, e := range evs {
		if e.Cancelled || e.Response.Response == "declined" {
			continue
		}
		start, err := time.Parse("2006-01-02T15:04:05.9999999", e.Start.DateTime)
		if err != nil {
			if start, err = time.Parse(time.RFC3339, e.Start.DateTime); err != nil {
				continue
			}
		}
		var who []string
		for _, a := range e.Attendees {
			if a.Email.Address != "" {
				who = append(who, strings.ToLower(a.Email.Address))
			}
		}
		out = append(out, ports.CalendarEvent{ID: e.ID, Title: e.Subject, Agenda: e.Body, Start: start.UTC(), Attendees: who})
	}
	return out, nil
}
