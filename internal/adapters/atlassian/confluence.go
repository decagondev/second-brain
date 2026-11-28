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

// Confluence is the docs: confluence connector (SB-6201; sb-connector(3)): the pages and
// blog posts of the spaces sb-cell(5) atlassian.spaces names, with their labels, as text,
// and each one's read restrictions as its ACL copy. The live check asks Confluence whether
// the person may read each page now. A space open to anonymous users makes its pages wide
// (SB-5401). Read-only.
type Confluence struct {
	Client *Client
	Spaces []string // space keys
	Dir    *Directory
	// Text reads a page's storage format (XHTML) as text.
	Text func(name string, b []byte) (string, bool)
}

const confluenceKind = "docs"

// ConfluenceUsers is the prefilter entry for a page any of the site's Confluence users may
// open; the live check decides.
const ConfluenceUsers = "confluence:users"

// ACLDeadline bounds one ACL batch; a late answer is Unknown.
const ACLDeadline = 800 * time.Millisecond

var _ ports.Connector = (*Confluence)(nil)

// APIVersion implements ports.Connector.
func (c *Confluence) APIVersion() int { return ports.ConnectorAPIVersion }

type spaceInfo struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

func (c *Confluence) space(ctx context.Context, key string) (spaceInfo, error) {
	var p struct {
		Results []spaceInfo `json:"results"`
	}
	if err := c.Client.get(ctx, "/wiki/api/v2/spaces?keys="+url.QueryEscape(key), &p); err != nil {
		return spaceInfo{}, err
	}
	for _, s := range p.Results {
		if s.Key == key {
			return s, nil
		}
	}
	return spaceInfo{}, fmt.Errorf("atlassian: no Confluence space %q", key)
}

type content struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	SpaceID string `json:"spaceId"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
}

// list is every page or blog post of a space, following the next links.
func (c *Confluence) list(ctx context.Context, spaceID, what string) ([]content, error) {
	var out []content
	next := "/wiki/api/v2/spaces/" + url.PathEscape(spaceID) + "/" + what + "?limit=250"
	for next != "" {
		var p struct {
			Results []content `json:"results"`
			Links   struct {
				Next string `json:"next"`
			} `json:"_links"`
		}
		if err := c.Client.get(ctx, next, &p); err != nil {
			return nil, err
		}
		out = append(out, p.Results...)
		next = p.Links.Next
	}
	return out, nil
}

// cursor is each item's version, by ref id ("page:ID", "blog:ID").
type confluenceCursor struct {
	Versions map[string]int `json:"versions"`
}

// Changes implements ports.Connector: every page and blog post of the named spaces is
// listed with its version; a new or newer one is a change, and one no longer listed is
// gone. The order is the spaces', then the site's, so a first sync resumes (SB-6004).
func (c *Confluence) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	var at confluenceCursor
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &at); err != nil {
			return nil, cur, fmt.Errorf("atlassian: a cursor that isn't ours: %w", err)
		}
	}
	next := confluenceCursor{Versions: map[string]int{}}
	var set ports.ChangeSet
	for _, key := range c.Spaces {
		sp, err := c.space(ctx, key)
		if err != nil {
			return nil, cur, err
		}
		for _, what := range []string{"pages", "blogposts"} {
			items, err := c.list(ctx, sp.ID, what)
			if err != nil {
				return nil, cur, err
			}
			pre := map[string]string{"pages": "page:", "blogposts": "blog:"}[what]
			for _, it := range items {
				id := pre + it.ID
				next.Versions[id] = it.Version.Number
				if at.Versions[id] != it.Version.Number {
					set = append(set, ports.Change{Ref: ports.SourceRef{Kind: confluenceKind, ID: id}})
				}
			}
		}
	}
	gone := make([]string, 0)
	for id := range at.Versions {
		if _, ok := next.Versions[id]; !ok {
			gone = append(gone, id)
		}
	}
	sort.Strings(gone)
	for _, id := range gone {
		set = append(set, ports.Change{Ref: ports.SourceRef{Kind: confluenceKind, ID: id}, Deleted: true})
	}
	b, _ := json.Marshal(next)
	return set, ports.Cursor(b), nil
}

func split(id string) (path, raw string, err error) {
	switch {
	case strings.HasPrefix(id, "page:"):
		return "pages", strings.TrimPrefix(id, "page:"), nil
	case strings.HasPrefix(id, "blog:"):
		return "blogposts", strings.TrimPrefix(id, "blog:"), nil
	}
	return "", "", fmt.Errorf("atlassian: %q isn't a Confluence ref", id)
}

type restrictions struct {
	Restrictions struct {
		User struct {
			Results []user `json:"results"`
		} `json:"user"`
		Group struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
		} `json:"group"`
	} `json:"restrictions"`
}

// Fetch implements ports.Connector: the page's text with its labels, its version, and its
// ACL copy: the read restrictions' people and groups, else anyone for a space open to
// anonymous users, else every Confluence user (the live check decides).
func (c *Confluence) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	what, id, err := split(ref.ID)
	if err != nil {
		return ports.Document{}, err
	}
	var it content
	if err := c.Client.get(ctx, "/wiki/api/v2/"+what+"/"+url.PathEscape(id)+"?body-format=storage", &it); err != nil {
		return ports.Document{}, err
	}
	var labels struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := c.Client.get(ctx, "/wiki/api/v2/"+what+"/"+url.PathEscape(id)+"/labels", &labels); err != nil {
		return ports.Document{}, err
	}
	var r restrictions
	if err := c.Client.get(ctx, "/wiki/rest/api/content/"+url.PathEscape(id)+"/restriction/byOperation/read", &r); err != nil {
		return ports.Document{}, err
	}
	var sp struct {
		Key string `json:"key"`
	}
	if err := c.Client.get(ctx, "/wiki/api/v2/spaces/"+url.PathEscape(it.SpaceID), &sp); err != nil {
		return ports.Document{}, err
	}
	var b strings.Builder
	if len(labels.Results) > 0 {
		var ls []string
		for _, l := range labels.Results {
			ls = append(ls, l.Name)
		}
		b.WriteString("Labels: " + strings.Join(ls, ", ") + "\n\n")
	}
	if c.Text != nil {
		t, _ := c.Text(it.Title+".html", []byte(it.Body.Storage.Value))
		b.WriteString(t)
	}
	var pre []string
	for _, u := range r.Restrictions.User.Results {
		if u.Email != "" {
			pre = append(pre, strings.ToLower(u.Email))
		}
	}
	for _, g := range r.Restrictions.Group.Results {
		pre = append(pre, "group:"+strings.ToLower(g.Name))
	}
	if len(pre) == 0 {
		open, err := c.anonymous(ctx, it.SpaceID)
		if err != nil {
			return ports.Document{}, err
		}
		pre = []string{ConfluenceUsers}
		if open {
			pre = []string{"anyone"} // wide (SB-5401)
		}
	}
	sort.Strings(pre)
	return ports.Document{Ref: ref, Version: it.Version.Number, Title: it.Title, Text: b.String(), ACLPrefilter: pre, Location: "/" + sp.Key + "/" + it.Title}, nil
}

// anonymous says a space lets anonymous users read it.
func (c *Confluence) anonymous(ctx context.Context, spaceID string) (bool, error) {
	var p struct {
		Results []struct {
			Principal struct {
				Type string `json:"type"`
			} `json:"principal"`
			Operation struct {
				Key string `json:"key"`
			} `json:"operation"`
		} `json:"results"`
	}
	if err := c.Client.get(ctx, "/wiki/api/v2/spaces/"+url.PathEscape(spaceID)+"/permissions", &p); err != nil {
		return false, err
	}
	for _, x := range p.Results {
		if x.Principal.Type == "anonymous" && x.Operation.Key == "read" {
			return true, nil
		}
	}
	return false, nil
}

// ACL implements ports.Connector: Confluence's own permission check for the person's
// account, each page, live. No account is Deny; anything else that fails, or is late, is
// Unknown (LAW-03).
func (c *Confluence) ACL(ctx context.Context, u ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	if u.Email == "" || c.Dir == nil {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	acct, err := c.Dir.AccountID(ctx, u.Email)
	if errors.Is(err, ErrNoUser) {
		for i := range out {
			out[i] = ports.VerdictDeny
		}
		return out, nil
	}
	if err != nil {
		return out, nil
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 8)
	for i, r := range refs {
		_, id, err := split(r.ID)
		if err != nil {
			continue
		}
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var res struct {
				HasPermission bool `json:"hasPermission"`
			}
			body := map[string]any{"subject": map[string]string{"type": "user", "identifier": acct}, "operation": "read"}
			if err := c.Client.do(ctx, "POST", "/wiki/rest/api/content/"+url.PathEscape(id)+"/permission/check", body, &res); err != nil {
				if notFound(err) {
					mu.Lock()
					out[i] = ports.VerdictDeny // gone: nobody may read it
					mu.Unlock()
				}
				return
			}
			v := ports.VerdictDeny
			if res.HasPermission {
				v = ports.VerdictAllow
			}
			mu.Lock()
			out[i] = v
			mu.Unlock()
		}(i, id)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil
	}
	return out, nil
}

// Act implements ports.Connector: Confluence is read-only.
func (c *Confluence) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, errors.New("atlassian: ENOTSUP: Confluence is read-only")
}
