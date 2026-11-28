package atlassian

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// Jira is the tickets: jira connector (SB-6204; sb-connector(3)): the issues of the projects
// sb-cell(5) atlassian.projects names, with their comments, labels and status (as Linear's
// are, SB-4204 and SB-4604: a Labels: and a Status: line first), and the people who may
// browse each one as its ACL copy. The live check asks Jira, issue by issue, whether the
// person may browse it now, which takes permission schemes and issue security levels into
// account. Read-only.
type Jira struct {
	Client *Client
	// Projects are project keys; Queues are service desk project keys read only for
	// request-received (SB-6206), never indexed.
	Projects []string
}

const jiraKind = "tickets"

// JiraUsers is the prefilter entry for an issue every Jira user may browse (a project
// open to the site, no security level); the live check decides.
const JiraUsers = "jira:users"

// MaxPrefilter is the most people an issue's ACL copy names; more is JiraUsers.
const MaxPrefilter = 1000

var _ ports.Connector = (*Jira)(nil)

// APIVersion implements ports.Connector.
func (j *Jira) APIVersion() int { return ports.ConnectorAPIVersion }

type issueRef struct {
	Key    string `json:"key"`
	Fields struct {
		Updated string `json:"updated"`
		Created string `json:"created"`
	} `json:"fields"`
}

// keys is every issue of projects with when it changed, in key order.
func (j *Jira) keys(ctx context.Context, projects []string) ([]issueRef, error) {
	if len(projects) == 0 {
		return nil, nil
	}
	var qs []string
	for _, p := range projects {
		qs = append(qs, `"`+strings.ReplaceAll(p, `"`, "")+`"`)
	}
	jql := "project in (" + strings.Join(qs, ",") + ") ORDER BY key"
	var out []issueRef
	token := ""
	for {
		u := "/rest/api/3/search/jql?fields=updated,created&maxResults=100&jql=" + url.QueryEscape(jql)
		if token != "" {
			u += "&nextPageToken=" + url.QueryEscape(token)
		}
		var p struct {
			Issues        []issueRef `json:"issues"`
			NextPageToken string     `json:"nextPageToken"`
			IsLast        bool       `json:"isLast"`
		}
		if err := j.Client.get(ctx, u, &p); err != nil {
			return nil, err
		}
		out = append(out, p.Issues...)
		if p.IsLast || p.NextPageToken == "" {
			return out, nil
		}
		token = p.NextPageToken
	}
}

type jiraCursor struct {
	Updated map[string]string `json:"updated"`
}

// Changes implements ports.Connector: every issue of the projects with its updated time; a
// new or changed one is a change, one no longer listed is gone.
func (j *Jira) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	var at jiraCursor
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &at); err != nil {
			return nil, cur, fmt.Errorf("atlassian: a cursor that isn't ours: %w", err)
		}
	}
	refs, err := j.keys(ctx, j.Projects)
	if err != nil {
		return nil, cur, err
	}
	next := jiraCursor{Updated: map[string]string{}}
	var set ports.ChangeSet
	for _, r := range refs {
		next.Updated[r.Key] = r.Fields.Updated
		if at.Updated[r.Key] != r.Fields.Updated {
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: jiraKind, ID: r.Key}})
		}
	}
	var gone []string
	for k := range at.Updated {
		if _, ok := next.Updated[k]; !ok {
			gone = append(gone, k)
		}
	}
	sort.Strings(gone)
	for _, k := range gone {
		set = append(set, ports.Change{Ref: ports.SourceRef{Kind: jiraKind, ID: k}, Deleted: true})
	}
	b, _ := json.Marshal(next)
	return set, ports.Cursor(b), nil
}

// Issue is one issue as the connector reads it.
type Issue struct {
	Key, Project, Summary, Text, Status string
	Done                                bool
	Labels                              []string
	Updated, Created                    time.Time
	Restricted                          bool // under an issue security level
}

type adfNode struct {
	Type    string    `json:"type"`
	Text    string    `json:"text"`
	Content []adfNode `json:"content"`
}

// adfText is an Atlassian Document Format tree as text, a block a line.
func adfText(n adfNode) string {
	if n.Type == "text" {
		return n.Text
	}
	var b strings.Builder
	for _, c := range n.Content {
		b.WriteString(adfText(c))
	}
	switch n.Type {
	case "paragraph", "heading", "listItem", "codeBlock", "blockquote":
		b.WriteString("\n")
	case "hardBreak":
		b.WriteString("\n")
	}
	return b.String()
}

// ReadIssue reads one issue whole.
func (j *Jira) ReadIssue(ctx context.Context, key string) (Issue, error) {
	var raw struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string   `json:"summary"`
			Description *adfNode `json:"description"`
			Labels      []string `json:"labels"`
			Updated     string   `json:"updated"`
			Created     string   `json:"created"`
			Project     struct {
				Key string `json:"key"`
			} `json:"project"`
			Status struct {
				Name           string `json:"name"`
				StatusCategory struct {
					Key string `json:"key"`
				} `json:"statusCategory"`
			} `json:"status"`
			Security *struct {
				ID string `json:"id"`
			} `json:"security"`
			Comment struct {
				Comments []struct {
					Body adfNode `json:"body"`
				} `json:"comments"`
			} `json:"comment"`
		} `json:"fields"`
	}
	if err := j.Client.get(ctx, "/rest/api/3/issue/"+url.PathEscape(key)+"?fields=summary,description,comment,labels,status,security,project,updated,created", &raw); err != nil {
		return Issue{}, err
	}
	f := raw.Fields
	is := Issue{Key: raw.Key, Project: f.Project.Key, Summary: f.Summary, Status: f.Status.Name, Done: f.Status.StatusCategory.Key == "done",
		Labels: f.Labels, Restricted: f.Security != nil}
	is.Updated, _ = time.Parse("2006-01-02T15:04:05.000-0700", f.Updated)
	is.Created, _ = time.Parse("2006-01-02T15:04:05.000-0700", f.Created)
	var b strings.Builder
	if f.Description != nil {
		b.WriteString(strings.TrimSpace(adfText(*f.Description)))
	}
	for _, c := range f.Comment.Comments {
		b.WriteString("\n\n" + strings.TrimSpace(adfText(c.Body)))
	}
	is.Text = b.String()
	return is, nil
}

// Fetch implements ports.Connector: the issue as Linear's issues read (labels and status
// first), its version its updated time, and its ACL copy the people Jira says may browse
// it (or every Jira user when that is more than MaxPrefilter and it has no security level).
func (j *Jira) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	is, err := j.ReadIssue(ctx, ref.ID)
	if err != nil {
		return ports.Document{}, err
	}
	var b strings.Builder
	if len(is.Labels) > 0 { // SB-4204, as Linear
		b.WriteString("Labels: " + strings.Join(is.Labels, ", ") + "\n")
	}
	if is.Status != "" { // SB-4604
		st := "Status: " + is.Status
		if is.Done {
			st += " (closed)"
		}
		b.WriteString(st + "\n")
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString(is.Text)
	var people []user
	if err := j.Client.get(ctx, "/rest/api/3/user/permission/search?permissions=BROWSE_PROJECTS&maxResults="+fmt.Sprint(MaxPrefilter+1)+"&issueKey="+url.QueryEscape(is.Key), &people); err != nil {
		return ports.Document{}, err
	}
	var pre []string
	for _, u := range people {
		if u.Active && u.Email != "" {
			pre = append(pre, strings.ToLower(u.Email))
		}
	}
	if len(pre) > MaxPrefilter && !is.Restricted {
		pre = []string{JiraUsers}
	}
	sort.Strings(pre)
	v := int(is.Updated.UnixMicro())
	if v <= 0 {
		return ports.Document{}, fmt.Errorf("atlassian: %s has no updated time", is.Key)
	}
	return ports.Document{Ref: ref, Version: v, Title: is.Key + " " + is.Summary, Text: b.String(), ACLPrefilter: pre, Location: is.Project}, nil
}

// ACL implements ports.Connector: Jira's own answer to whether the person may browse each
// issue now. No account is Deny; anything else that fails, or is late, is Unknown.
func (j *Jira) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	if u.Email == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 8)
	for i, r := range refs {
		wg.Add(1)
		go func(i int, key string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var people []user
			err := j.Client.get(ctx, "/rest/api/3/user/permission/search?permissions=BROWSE_PROJECTS&issueKey="+url.QueryEscape(key)+"&query="+url.QueryEscape(u.Email), &people)
			if notFound(err) {
				mu.Lock()
				out[i] = ports.VerdictDeny
				mu.Unlock()
				return
			}
			if err != nil {
				return
			}
			v := ports.VerdictDeny
			for _, p := range people {
				if strings.EqualFold(p.Email, u.Email) && p.Active {
					v = ports.VerdictAllow
				}
			}
			mu.Lock()
			out[i] = v
			mu.Unlock()
		}(i, r.ID)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil
	}
	return out, nil
}

// Act implements ports.Connector: the brain files nothing in Jira (SB-6207: a person does,
// from a draft).
func (j *Jira) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, errors.New("atlassian: ENOTSUP: the brain never writes to Jira")
}

// Requests are the issues of a service desk project created after since, oldest first, and
// the newest created time seen (SB-6206). Only a queue a space names is ever read.
func (j *Jira) Requests(ctx context.Context, project string, since time.Time) ([]Issue, time.Time, error) {
	refs, err := j.keys(ctx, []string{project})
	if err != nil {
		return nil, since, err
	}
	var out []Issue
	newest := since
	for _, r := range refs {
		c, err := time.Parse("2006-01-02T15:04:05.000-0700", r.Fields.Created)
		if err != nil || !c.After(since) {
			continue
		}
		is, err := j.ReadIssue(ctx, r.Key)
		if err != nil {
			return nil, since, err
		}
		out = append(out, is)
		if c.After(newest) {
			newest = c
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Created.Before(out[b].Created) })
	return out, newest, nil
}
