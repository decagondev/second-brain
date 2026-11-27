package msgraph

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/core/ports"
)

// SharePoint and OneDrive, `docs: sharepoint` (sb-connector(3), SharePoint and OneDrive;
// SB-5901, SB-5902, SB-5906): the drives sb-cell(5) names, read with delta queries from a
// stored cursor; each item's live permissions, sharing links included, become its ACL copy;
// its text comes through the Office readers with pages and slides; its Purview label maps
// to the four levels, and an unmapped one counts as restricted (LAW-03). Teams meeting
// transcripts of the organizers it names are documents too, open only to the meeting's
// organizer and attendees, who could open its recording.

// The connector's limits.
const (
	ACLDeadline    = 800 * time.Millisecond
	ACLConcurrency = 8
	MaxContent     = 25 << 20
	kind           = "docs"
)

// SharePoint is one tenant's SharePoint and OneDrive.
type SharePoint struct {
	Graph  *Client
	Drives []string // drive ids: a site's document library or a OneDrive
	// Domain is the company's email domain: an organization sharing link grants it.
	Domain string
	// Labels maps Purview label ids to public, internal, confidential or restricted
	// (sb-cell(5) labels.microsoft; SB-5902).
	Labels map[string]string
	// Organizers are the Entra user ids whose Teams meetings' transcripts are read (SB-5906).
	Organizers []string
	// Dir expands a person's groups for the live check (SB-5903); nil grants by group never.
	Dir *Directory
	// Readers read a file's text, pages and review date, and a transcript's text: the
	// folder connector's Office readers, which the cell hands in (an adapter imports no
	// sibling adapter). A nil reader reads nothing of that kind.
	Readers Readers

	mu       sync.Mutex
	unmapped map[string]int
}

var _ ports.Connector = (*SharePoint)(nil)

// ErrNotSupported is Act on this read-only connector.
var ErrNotSupported = errors.New("ENOTSUP: sharepoint connector is read-only")

// APIVersion implements ports.Connector.
func (c *SharePoint) APIVersion() int { return ports.ConnectorAPIVersion }

// Unmapped is the Purview label ids the last syncs met that sb-cell(5) doesn't map, by how
// many items carried each (SB-5902): ids and counts only, never which documents.
func (c *SharePoint) Unmapped() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]int{}
	for k, v := range c.unmapped {
		out[k] = v
	}
	return out
}

// An item's id carries its drive or meeting, encoded so it is one cite-safe token.
func spID(drive, item string) string {
	return "sp-" + base64.RawURLEncoding.EncodeToString([]byte(drive+"|"+item))
}

// ItemRef is the connector's ref for item of drive.
func ItemRef(drive, item string) ports.SourceRef {
	return ports.SourceRef{Kind: kind, ID: spID(drive, item)}
}

func tmID(org, meeting, transcript string) string {
	return "tm-" + base64.RawURLEncoding.EncodeToString([]byte(org+"|"+meeting+"|"+transcript))
}

func parseID(id string) (string, []string, error) {
	pre, rest, ok := strings.Cut(id, "-")
	if !ok || (pre != "sp" && pre != "tm") {
		return "", nil, fmt.Errorf("msgraph: %q isn't a SharePoint or Teams id", id)
	}
	b, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return "", nil, fmt.Errorf("msgraph: %q isn't a SharePoint or Teams id", id)
	}
	parts := strings.Split(string(b), "|")
	if (pre == "sp" && len(parts) != 2) || (pre == "tm" && len(parts) != 3) {
		return "", nil, fmt.Errorf("msgraph: %q isn't a SharePoint or Teams id", id)
	}
	return pre, parts, nil
}

type driveItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	CTag    string `json:"cTag"`
	Size    int    `json:"size"`
	Deleted *struct {
		State string `json:"state"`
	} `json:"deleted"`
	Folder *struct{} `json:"folder"`
	File   *struct{} `json:"file"`
	Parent struct {
		Path string `json:"path"`
	} `json:"parentReference"`
	CreatedBy struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	} `json:"createdBy"`
	SensitivityLabel *struct {
		LabelID string `json:"labelId"`
	} `json:"sensitivityLabel"`
	// SharedChanged marks a delta entry whose sharing changed.
	SharedChanged string `json:"@microsoft.graph.sharedChanged"`
}

// sharingChanges asks a drive's delta to mark the items whose sharing changed.
const sharingChanges = "deltashowsharingchanges"

type permission struct {
	GrantedToV2 *struct {
		User *struct {
			Email string `json:"email"`
		} `json:"user"`
		Group *struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"group"`
	} `json:"grantedToV2"`
	Link *struct {
		Scope string `json:"scope"`
	} `json:"link"`
}

type cursor struct {
	Drives   map[string]string `json:"drives,omitempty"`
	Meetings map[string]string `json:"meetings,omitempty"`
}

// Changes implements ports.Connector: each drive's delta and each organizer's transcripts
// from the links the cursor keeps, or everything for an empty cursor. A link Graph no
// longer takes (410) starts that drive over.
func (c *SharePoint) Changes(ctx context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	var at cursor
	if cur != "" {
		if err := json.Unmarshal([]byte(cur), &at); err != nil {
			return nil, cur, fmt.Errorf("msgraph: a cursor that isn't ours: %w", err)
		}
	}
	next := cursor{Drives: map[string]string{}, Meetings: map[string]string{}}
	var set ports.ChangeSet
	for _, d := range c.Drives {
		link := at.Drives[d]
		if link == "" {
			link = "/drives/" + url.PathEscape(d) + "/root/delta"
		}
		items, delta, _, err := allWith[driveItem](ctx, c.Graph, link, sharingChanges, 0)
		var ge *Error
		if errors.As(err, &ge) && ge.Status == 410 {
			items, delta, _, err = allWith[driveItem](ctx, c.Graph, "/drives/"+url.PathEscape(d)+"/root/delta", sharingChanges, 0)
		}
		if err != nil {
			return nil, cur, err
		}
		for _, it := range items {
			ref := ports.SourceRef{Kind: kind, ID: spID(d, it.ID)}
			switch {
			case it.Deleted != nil:
				set = append(set, ports.Change{Ref: ref, Deleted: true})
			case it.Folder != nil || it.File == nil:
			default: // a sharing change re-reads the item whole, which also covers a content change
				set = append(set, ports.Change{Ref: ref, ACLOnly: strings.EqualFold(it.SharedChanged, "true")})
			}
		}
		next.Drives[d] = delta
	}
	for _, o := range c.Organizers {
		link := at.Meetings[o]
		if link == "" {
			link = "/users/" + url.PathEscape(o) + "/onlineMeetings/getAllTranscripts(meetingOrganizerUserId='" + url.PathEscape(o) + "')/delta"
		}
		ts, delta, _, err := all[struct {
			ID        string `json:"id"`
			MeetingID string `json:"meetingId"`
		}](ctx, c.Graph, link, 0)
		if err != nil {
			return nil, cur, err
		}
		for _, t := range ts {
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: tmID(o, t.MeetingID, t.ID)}})
		}
		next.Meetings[o] = delta
	}
	b, _ := json.Marshal(next)
	return set, ports.Cursor(b), nil
}

// version is a file's content version, the number after the last comma of its cTag.
func version(ctag string) (int, error) {
	t := strings.Trim(ctag, `"`)
	i := strings.LastIndex(t, ",")
	if i < 0 {
		return 0, errors.New("msgraph: a cTag without a version")
	}
	return strconv.Atoi(t[i+1:])
}

// Fetch implements ports.Connector.
func (c *SharePoint) Fetch(ctx context.Context, ref ports.SourceRef) (ports.Document, error) {
	pre, parts, err := parseID(ref.ID)
	if err != nil {
		return ports.Document{}, err
	}
	if pre == "tm" {
		return c.fetchTranscript(ctx, ref, parts[0], parts[1], parts[2])
	}
	d, id := parts[0], parts[1]
	base := "/drives/" + url.PathEscape(d) + "/items/" + url.PathEscape(id)
	var it driveItem
	if err := c.Graph.get(ctx, base, &it); err != nil {
		return ports.Document{}, err
	}
	if it.Deleted != nil || it.File == nil {
		return ports.Document{}, fmt.Errorf("msgraph: %s is not a file now", ref.ID)
	}
	v, err := version(it.CTag)
	if err != nil {
		return ports.Document{}, err
	}
	perms, err := c.permissions(ctx, d, id)
	if err != nil {
		return ports.Document{}, err
	}
	doc := ports.Document{Ref: ref, Version: v, Title: it.Name, ACLPrefilter: c.prefilter(perms), Location: location(it), Owner: strings.ToLower(it.CreatedBy.User.Email)}
	if it.SensitivityLabel != nil {
		doc.Label = c.label(it.SensitivityLabel.LabelID)
	}
	if it.Size <= MaxContent && c.Readers.Text != nil {
		b, err := c.Graph.raw(ctx, base+"/content")
		if err != nil {
			return ports.Document{}, err
		}
		doc.Text, _ = c.Readers.Text(it.Name, b)
		if c.Readers.Pages != nil {
			doc.Pages = c.Readers.Pages(it.Name, b) // SB-5205: pages and slides
		}
		if e := strings.ToLower(path.Ext(it.Name)); c.Readers.ReviewBy != nil && (e == ".docx" || e == ".pptx") {
			doc.ReviewBy = c.Readers.ReviewBy(b)
		}
		if c.Readers.Images != nil { // SB-6101, SB-6102: scans, pictures and recordings
			doc.Images, doc.Media = c.Readers.Images(it.Name, b), c.Readers.Media(it.Name, b)
		}
	}
	return doc, nil
}

// Readers are the text readers the connector uses.
type Readers struct {
	// Images and Media are a file's pictures and recordings (SB-6101, SB-6102); both or
	// neither.
	Images, Media func(name string, b []byte) []ports.Part
	Text          func(name string, b []byte) (string, bool)
	Pages         func(name string, b []byte) []int
	ReviewBy      func(b []byte) string
	Transcript    func(b []byte) (string, int)
}

// location is the item's folder under its drive's root, then its name.
func location(it driveItem) string {
	p := it.Parent.Path
	if i := strings.Index(p, "root:"); i >= 0 {
		p = p[i+len("root:"):]
	} else {
		p = ""
	}
	return strings.TrimSuffix("/"+strings.Trim(p, "/"), "/") + "/" + it.Name
}

func (c *SharePoint) permissions(ctx context.Context, drive, id string) ([]permission, error) {
	ps, _, _, err := all[permission](ctx, c.Graph, "/drives/"+url.PathEscape(drive)+"/items/"+url.PathEscape(id)+"/permissions", 0)
	return ps, err
}

// prefilter is the ACL copy: emails, group:EMAIL (or group:ID without a mail), anyone for
// an anonymous link and domain:DOMAIN for an organization link, both wide (SB-5401).
func (c *SharePoint) prefilter(ps []permission) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, p := range ps {
		switch {
		case p.Link != nil && p.Link.Scope == "anonymous":
			add("anyone")
		case p.Link != nil && p.Link.Scope == "organization" && c.Domain != "":
			add("domain:" + strings.ToLower(c.Domain))
		case p.GrantedToV2 != nil && p.GrantedToV2.User != nil:
			add(strings.ToLower(p.GrantedToV2.User.Email))
		case p.GrantedToV2 != nil && p.GrantedToV2.Group != nil:
			g := p.GrantedToV2.Group
			if g.Email != "" {
				add("group:" + strings.ToLower(g.Email))
			} else {
				add("group:" + strings.ToLower(g.ID))
			}
		}
	}
	sort.Strings(out)
	return out
}

// label is a Purview label's level; one sb-cell(5) doesn't map counts as restricted until
// an admin maps it, and is counted for the health report.
func (c *SharePoint) label(id string) string {
	if id == "" {
		return ""
	}
	if l, ok := c.Labels[id]; ok {
		return l
	}
	c.mu.Lock()
	if c.unmapped == nil {
		c.unmapped = map[string]int{}
	}
	c.unmapped[id]++
	c.mu.Unlock()
	return "restricted"
}

type meeting struct {
	Subject      string `json:"subject"`
	Start        string `json:"startDateTime"`
	Participants struct {
		Organizer struct {
			UPN string `json:"upn"`
		} `json:"organizer"`
		Attendees []struct {
			UPN string `json:"upn"`
		} `json:"attendees"`
	} `json:"participants"`
}

func (m meeting) people() []string {
	out := []string{strings.ToLower(m.Participants.Organizer.UPN)}
	for _, a := range m.Participants.Attendees {
		if a.UPN != "" {
			out = append(out, strings.ToLower(a.UPN))
		}
	}
	sort.Strings(out)
	return out
}

func meetingPath(org, id string) string {
	return "/users/" + url.PathEscape(org) + "/onlineMeetings/" + url.PathEscape(id)
}

// fetchTranscript is a Teams meeting's transcript as a document of type transcript
// (SB-3701): "SUBJECT (DATE) - Transcript", speakers and times kept, open to the organizer
// and attendees. A meeting without a transcript is never listed, so never fetched.
func (c *SharePoint) fetchTranscript(ctx context.Context, ref ports.SourceRef, org, mid, tid string) (ports.Document, error) {
	var m meeting
	if err := c.Graph.get(ctx, meetingPath(org, mid), &m); err != nil {
		return ports.Document{}, err
	}
	b, err := c.Graph.raw(ctx, meetingPath(org, mid)+"/transcripts/"+url.PathEscape(tid)+"/content?$format=text/vtt")
	if err != nil {
		return ports.Document{}, err
	}
	text := ""
	if c.Readers.Transcript != nil {
		text, _ = c.Readers.Transcript(b)
	}
	day := m.Start
	if len(day) >= 10 {
		day = day[:10]
	}
	title := strings.TrimSpace(m.Subject) + " (" + day + ") - Transcript"
	return ports.Document{Ref: ref, Version: 1, Title: title, Text: text, ACLPrefilter: m.people(), Location: "/Teams meetings/" + title,
		Owner: strings.ToLower(m.Participants.Organizer.UPN)}, nil
}

// ACL implements ports.Connector: each item's permissions read again now, and the person's
// groups from Entra, within ACLDeadline. An item that is gone, a read that fails and a late
// answer are Unknown (LAW-03).
func (c *SharePoint) ACL(ctx context.Context, user ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	if user.Email == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	email := strings.ToLower(user.Email)
	var groups map[string]bool
	groupsOf := func() (map[string]bool, error) {
		if groups != nil || c.Dir == nil {
			return groups, nil
		}
		gs, err := c.Dir.GroupsOf(ctx, email)
		if err != nil {
			return nil, err
		}
		groups = map[string]bool{}
		for _, g := range gs {
			groups[g] = true
		}
		return groups, nil
	}
	var once sync.Once
	var gs map[string]bool
	var gerr error
	shared := func() (map[string]bool, error) { // the person's groups, read once for the batch
		once.Do(func() { gs, gerr = groupsOf() })
		return gs, gerr
	}
	sem := make(chan struct{}, ACLConcurrency)
	var wg sync.WaitGroup
	for i, r := range refs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = c.verdict(ctx, email, id, shared)
		}(i, r.ID)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return make([]ports.Verdict, len(refs)), nil // late is Unknown
	}
	return out, nil
}

func (c *SharePoint) verdict(ctx context.Context, email, id string, groupsOf func() (map[string]bool, error)) ports.Verdict {
	pre, parts, err := parseID(id)
	if err != nil {
		return ports.VerdictUnknown
	}
	if pre == "tm" {
		var m meeting
		if err := c.Graph.get(ctx, meetingPath(parts[0], parts[1]), &m); err != nil {
			return ports.VerdictUnknown
		}
		for _, p := range m.people() {
			if p == email {
				return ports.VerdictAllow
			}
		}
		return ports.VerdictDeny
	}
	ps, err := c.permissions(ctx, parts[0], parts[1])
	if err != nil {
		return ports.VerdictUnknown // gone or unreadable
	}
	for _, t := range c.prefilter(ps) {
		switch {
		case t == "anyone", t == email:
			return ports.VerdictAllow
		case strings.HasPrefix(t, "domain:") && strings.HasSuffix(email, "@"+strings.TrimPrefix(t, "domain:")):
			return ports.VerdictAllow
		case strings.HasPrefix(t, "group:"):
			gs, err := groupsOf()
			if err != nil {
				return ports.VerdictUnknown
			}
			if gs[t] {
				return ports.VerdictAllow
			}
		}
	}
	return ports.VerdictDeny
}

// Act is not supported: read-only.
func (c *SharePoint) Act(context.Context, ports.Action, ports.IdempotencyKey) (ports.Receipt, error) {
	return ports.Receipt{}, ErrNotSupported
}
