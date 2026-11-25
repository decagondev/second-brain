package ask

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Projects (sb-spaces(7), Projects; SB-4803, SB-4804): Start a project makes a collection
// for the lead and members, a goal draft and a decision log, and starts the kickoff rhythm,
// whose first step is a brief of similar past projects drawn only from what the lead can
// open. Close project archives the collection read-only and starts the close-out rhythm,
// whose retrospective proposes lessons the space's stewards may publish as notes.

// The projects' limits.
const (
	MaxProjectName      = 80
	MaxProjectMembers   = 20
	MaxProjectDecisions = 50
	MaxLessons          = 5
	MaxDecisionText     = 300
	MaxSimilarProjects  = 5
)

// The states of a project.
const (
	ProjectOpen   = "open"
	ProjectClosed = "closed"
)

// ProjectRef marks a queue item a project sent: a member to approve or a lesson to publish.
type ProjectRef struct {
	Project string `json:"project"`
	Kind    string `json:"kind"` // member or lesson
	Email   string `json:"email,omitempty"`
	Lesson  int    `json:"lesson,omitempty"`
}

// ProjectDecision is one line of a project's decision log, citing the document it rests on.
type ProjectDecision struct {
	Text string    `json:"text"`
	Cite string    `json:"cite"` // KIND:ID
	By   string    `json:"by"`
	At   time.Time `json:"at"`
}

// Lesson is one lesson the lead proposed at close-out, with the document it comes from and
// the item it waits in for the stewards.
type Lesson struct {
	Text string `json:"text"`
	Cite string `json:"cite"`
	Item string `json:"item,omitempty"`
}

// PendingMember is a member from another space, waiting for their steward to approve.
type PendingMember struct {
	Email string `json:"email"`
	Space string `json:"space"`
	Item  string `json:"item"`
}

// Project is one project as the cell keeps it.
type Project struct {
	ID         string            `json:"id"`
	Tenant     string            `json:"tenant"`
	Space      string            `json:"space"`
	Name       string            `json:"name"`
	Lead       string            `json:"lead"` // a user id
	Collection string            `json:"collection"`
	Pending    []PendingMember   `json:"pending,omitempty"`
	Goal       string            `json:"goal"`
	Decisions  []ProjectDecision `json:"decisions,omitempty"`
	Lessons    []Lesson          `json:"lessons,omitempty"`
	State      string            `json:"state"`
	At         time.Time         `json:"at"`
	Closed     *time.Time        `json:"closed,omitempty"`
	Kickoff    string            `json:"kickoff,omitempty"`
	CloseOut   string            `json:"close_out,omitempty"`
}

// ProjectStore keeps projects.
type ProjectStore interface {
	PutProject(ctx context.Context, p Project) error
	GetProject(ctx context.Context, tenant, id string) (Project, bool, error)
	Projects(ctx context.Context, tenant string) ([]Project, error)
}

// ProjectInput is POST /v1/spaces/NAME/projects.
type ProjectInput struct {
	Name      string   `json:"name"`
	Members   []string `json:"members"`   // emails
	Documents []string `json:"documents"` // KIND:ID
}

// DecisionInput is POST /v1/projects/ID/decisions, and a lesson at close-out.
type DecisionInput struct {
	Text string `json:"text"`
	Cite string `json:"cite"`
}

// CloseInput is POST /v1/projects/ID/close.
type CloseInput struct {
	Lessons []DecisionInput `json:"lessons"`
}

// ProjectView is a project as one reader sees it: decisions only where they can open the
// cite, the rest counted.
type ProjectView struct {
	ID         string            `json:"id"`
	Space      string            `json:"space"`
	Name       string            `json:"name"`
	Lead       bool              `json:"lead"`
	State      string            `json:"state"`
	Collection CollectionView    `json:"collection"`
	Goal       string            `json:"goal,omitempty"` // the lead's goal draft
	Decisions  []ProjectDecision `json:"decisions"`
	Hidden     int               `json:"hidden"`
	Pending    []string          `json:"pending,omitempty"` // the lead sees who waits
	Kickoff    string            `json:"kickoff,omitempty"`
	CloseOut   string            `json:"close_out,omitempty"`
}

func (x Rhythms) collections() CollectionService {
	c := x.Collections
	c.Agent = x.Stewards.Agent
	if c.NewID == nil {
		c.NewID, c.Now, c.Audit = x.Stewards.NewID, x.Stewards.Now, x.Stewards.Audit
	}
	if c.Store == nil {
		c.Store = x.Stewards.Agent.Collections
	}
	return c
}

func (x Rhythms) projectAudit(ctx context.Context, actor string, p Project, state, why string) error {
	if _, err := x.Stewards.Audit.Write(ctx, audit.Event{Actor: actor, Kind: "access", Subject: p.ID, State: state,
		Reason: "project: " + why, Space: auditSpace(SpaceKey(p.Space))}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

func (x Rhythms) putProject(ctx context.Context, p Project) error {
	if err := x.Projects.PutProject(ctx, p); err != nil {
		return fmt.Errorf("%w: projects: %v", ErrUnavailable, err)
	}
	return nil
}

// quarterOf is a time's quarter, YYYY-Qn.
func quarterOf(t time.Time) string {
	return fmt.Sprintf("%d-Q%d", t.Year(), (int(t.Month())-1)/3+1)
}

// StartProject starts a project in space with s as its lead: a collection shared with the
// members who are in the space, a request to their steward for each member from another
// space, a goal draft, an empty decision log, and the kickoff rhythm (SB-4803).
func (x Rhythms) StartProject(ctx context.Context, s tenancy.Scope, space string, in ProjectInput) (ProjectView, error) {
	if x.Projects == nil || x.ScopeIn == nil {
		return ProjectView{}, fmt.Errorf("%w: this cell keeps no projects", ErrInvalid)
	}
	if err := s.Check(); err != nil || s.Tenant() != x.Stewards.Agent.Tenant {
		return ProjectView{}, fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	ls, err := x.ScopeIn(ctx, s.User(), space)
	if err != nil {
		return ProjectView{}, fmt.Errorf("%w: you aren't a member of %s", ErrNotMember, space)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > MaxProjectName || strings.ContainsAny(name, "\n\r") {
		return ProjectView{}, fmt.Errorf("%w: a project's name is 1 to %d characters on one line", ErrInvalid, MaxProjectName)
	}
	if len(in.Members) > MaxProjectMembers {
		return ProjectView{}, fmt.Errorf("%w: at most %d members", ErrInvalid, MaxProjectMembers)
	}
	d, err := x.def(space, "kickoff")
	if err != nil {
		return ProjectView{}, err
	}
	var here []string
	var away []PendingMember
	for _, e := range in.Members {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		id, ok, err := x.Stewards.Owner(ctx, e)
		if err != nil {
			return ProjectView{}, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
		}
		if !ok {
			return ProjectView{}, fmt.Errorf("%w: %s isn't one active person", ErrInvalid, e)
		}
		if _, err := x.ScopeIn(ctx, id, space); err == nil {
			here = append(here, e)
			continue
		}
		home := x.homeOf(ctx, id)
		if home == "" {
			return ProjectView{}, fmt.Errorf("%w: %s is a member of no space", ErrInvalid, e)
		}
		away = append(away, PendingMember{Email: e, Space: home})
	}
	cv, err := x.collections().Create(ctx, ls, CollectionInput{Name: "Project: " + name, Items: in.Documents, Shared: here})
	if err != nil {
		return ProjectView{}, err
	}
	now := x.now()
	p := Project{ID: x.Stewards.NewID("pj_"), Tenant: x.Stewards.Agent.Tenant, Space: space, Name: name, Lead: s.User(),
		Collection: cv.ID, State: ProjectOpen, At: now}
	lead := s.User()
	if es, err := x.Stewards.Agent.emailsOf(ctx, s.User()); err == nil && len(es) > 0 {
		lead = es[0]
	}
	p.Goal = fmt.Sprintf("---\ntype: goal\nowner: %s\nquarter: %s\nstatus: on track\n---\n\n# %s\n\nWhat done looks like:\n", lead, quarterOf(now), name)
	c, _, err := x.Stewards.Agent.Collections.GetCollection(ctx, p.Tenant, cv.ID)
	if err != nil {
		return ProjectView{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	c.Project = p.ID
	if err := x.Stewards.Agent.Collections.PutCollection(ctx, c); err != nil {
		return ProjectView{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	for _, m := range away {
		sq := StewardQuestion{ID: x.Stewards.NewID("qs_"), Tenant: p.Tenant, Asker: s.User(), Space: SpaceKey(m.Space), At: now,
			Question: firstLine(fmt.Sprintf("Add %s to the project %s in %s? They would see only what they can already open.", m.Email, name, space)),
			Project:  &ProjectRef{Project: p.ID, Kind: "member", Email: m.Email}}
		if err := x.Stewards.Store.PutStewardQuestion(ctx, sq); err != nil {
			return ProjectView{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		m.Item = sq.ID
		p.Pending = append(p.Pending, m)
	}
	if err := x.projectAudit(ctx, s.User(), p, "project_started", fmt.Sprintf("%d members, %d waiting for their steward", len(here), len(away))); err != nil {
		return ProjectView{}, err
	}
	if err := x.putProject(ctx, p); err != nil {
		return ProjectView{}, err
	}
	r, err := x.start(ctx, s.User(), space, "kickoff", d, p.Collection)
	if err != nil {
		return ProjectView{}, err
	}
	p.Kickoff = r.ID
	if err := x.putProject(ctx, p); err != nil {
		return ProjectView{}, err
	}
	return x.projectView(ctx, s, p)
}

// homeOf is the first space, by name, a person is a member of now.
func (x Rhythms) homeOf(ctx context.Context, user string) string {
	if x.Spaces == nil {
		return ""
	}
	names := append([]string{}, x.Spaces()...)
	sort.Strings(names)
	for _, n := range names {
		if _, err := x.ScopeIn(ctx, user, n); err == nil {
			return n
		}
	}
	return ""
}

func (x Rhythms) project(ctx context.Context, id string) (Project, error) {
	if x.Projects == nil {
		return Project{}, fmt.Errorf("%w: this cell keeps no projects", ErrInvalid)
	}
	p, ok, err := x.Projects.GetProject(ctx, x.Stewards.Agent.Tenant, strings.TrimSpace(id))
	if err != nil {
		return Project{}, fmt.Errorf("%w: projects: %v", ErrUnavailable, err)
	}
	if !ok || p.Tenant != x.Stewards.Agent.Tenant {
		return Project{}, fmt.Errorf("%w: no such project", ErrInvalid)
	}
	return p, nil
}

// readProject is a project s may read: they lead it or its collection is shared with them.
func (x Rhythms) readProject(ctx context.Context, s tenancy.Scope, id string) (Project, error) {
	p, err := x.project(ctx, id)
	if err != nil {
		return Project{}, err
	}
	if _, err := x.Stewards.Agent.readCollection(ctx, s, x.Stewards.Agent.Collections, p.Collection); err != nil {
		return Project{}, fmt.Errorf("%w: no such project", ErrInvalid)
	}
	return p, nil
}

// Project is one project as s sees it.
func (x Rhythms) Project(ctx context.Context, s tenancy.Scope, id string) (ProjectView, error) {
	p, err := x.readProject(ctx, s, id)
	if err != nil {
		return ProjectView{}, err
	}
	return x.projectView(ctx, s, p)
}

func (x Rhythms) projectView(ctx context.Context, s tenancy.Scope, p Project) (ProjectView, error) {
	c, err := x.Stewards.Agent.readCollection(ctx, s, x.Stewards.Agent.Collections, p.Collection)
	if err != nil {
		return ProjectView{}, err
	}
	cv, err := x.collections().view(ctx, s, c, nil)
	if err != nil {
		return ProjectView{}, err
	}
	v := ProjectView{ID: p.ID, Space: p.Space, Name: p.Name, Lead: p.Lead == s.User(), State: p.State, Collection: cv, Decisions: []ProjectDecision{},
		Kickoff: p.Kickoff, CloseOut: p.CloseOut}
	if v.Lead {
		v.Goal = p.Goal
		for _, m := range p.Pending {
			v.Pending = append(v.Pending, m.Email)
		}
	}
	v.Decisions, v.Hidden, err = x.openDecisions(ctx, s, p.Decisions)
	return v, err
}

// openDecisions are the decisions whose cite s can open now, and how many others there are.
func (x Rhythms) openDecisions(ctx context.Context, s tenancy.Scope, ds []ProjectDecision) ([]ProjectDecision, int, error) {
	var cites []string
	for _, d := range ds {
		cites = append(cites, d.Cite+"@0")
	}
	open, err := x.Stewards.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return nil, 0, err
	}
	out, hidden := []ProjectDecision{}, 0
	for _, d := range ds {
		if !open[d.Cite+"@0"] {
			hidden++
			continue
		}
		d.By = "" // who wrote the line is the project's, not the reader's, business
		out = append(out, d)
	}
	return out, hidden, nil
}

func (x Rhythms) cleanCited(ctx context.Context, s tenancy.Scope, in DecisionInput) (DecisionInput, error) {
	in.Text, in.Cite = strings.TrimSpace(in.Text), strings.TrimSpace(in.Cite)
	if doc, _, ok := strings.Cut(in.Cite, "@"); ok {
		in.Cite = doc
	}
	if in.Text == "" || utf8.RuneCountInString(in.Text) > MaxDecisionText || strings.ContainsAny(in.Text, "\n\r") {
		return in, fmt.Errorf("%w: a line is 1 to %d characters on one line", ErrInvalid, MaxDecisionText)
	}
	if !strings.Contains(in.Cite, ":") {
		return in, fmt.Errorf("%w: every line cites a document, KIND:ID", ErrInvalid)
	}
	open, err := x.Stewards.Agent.opensCites(ctx, s, []string{in.Cite + "@0"})
	if err != nil {
		return in, err
	}
	if !open[in.Cite+"@0"] {
		return in, fmt.Errorf("%w: cite only a document you can open", ErrInvalid)
	}
	return in, nil
}

// Decide adds one line to an open project's decision log, as its lead or a member.
func (x Rhythms) Decide(ctx context.Context, s tenancy.Scope, id string, in DecisionInput) (ProjectView, error) {
	p, err := x.readProject(ctx, s, id)
	if err != nil {
		return ProjectView{}, err
	}
	if p.State != ProjectOpen {
		return ProjectView{}, fmt.Errorf("%w: the project is closed", ErrInvalid)
	}
	if len(p.Decisions) >= MaxProjectDecisions {
		return ProjectView{}, fmt.Errorf("%w: at most %d decisions", ErrInvalid, MaxProjectDecisions)
	}
	if in, err = x.cleanCited(ctx, s, in); err != nil {
		return ProjectView{}, err
	}
	p.Decisions = append(p.Decisions, ProjectDecision{Text: in.Text, Cite: in.Cite, By: s.User(), At: x.now()})
	if err := x.projectAudit(ctx, s.User(), p, "project_decided", "a decision logged"); err != nil {
		return ProjectView{}, err
	}
	if err := x.putProject(ctx, p); err != nil {
		return ProjectView{}, err
	}
	return x.projectView(ctx, s, p)
}

// AddMember approves a member from another space, as a steward of that member's space: the
// collection is shared with them, and they see only what they can already open (SB-4803).
func (x Rhythms) AddMember(ctx context.Context, s tenancy.Scope, id, email string) error {
	p, err := x.project(ctx, id)
	if err != nil {
		return err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	for i, m := range p.Pending {
		if m.Email != email {
			continue
		}
		if _, _, err := x.Stewards.steward(ctx, s, m.Space); err != nil {
			return err
		}
		c, ok, err := x.Stewards.Agent.Collections.GetCollection(ctx, p.Tenant, p.Collection)
		if err != nil {
			return fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
		}
		if !ok || c.Closed != nil {
			return fmt.Errorf("%w: the project is closed", ErrInvalid)
		}
		if !contains(c.Shared, email) {
			c.Shared = append(c.Shared, email)
		}
		if err := x.Stewards.Agent.Collections.PutCollection(ctx, c); err != nil {
			return fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
		}
		x.closeItem(ctx, m.Space, m.Item, s.User())
		p.Pending = append(p.Pending[:i:i], p.Pending[i+1:]...)
		if err := x.projectAudit(ctx, s.User(), p, "project_member_added", "a member from "+m.Space+" approved by their steward"); err != nil {
			return err
		}
		return x.putProject(ctx, p)
	}
	return fmt.Errorf("%w: nobody by that address waits to join the project", ErrInvalid)
}

// CloseProject closes a project, as its lead: the collection is archived read-only, the
// kickoff rhythm stops, each lesson waits in the space's queue as a note draft for its
// stewards, and the close-out rhythm drafts the retrospective (SB-4804). Nothing is
// deleted.
func (x Rhythms) CloseProject(ctx context.Context, s tenancy.Scope, id string, in CloseInput) (ProjectView, error) {
	p, err := x.readProject(ctx, s, id)
	if err != nil {
		return ProjectView{}, err
	}
	if p.Lead != s.User() {
		return ProjectView{}, fmt.Errorf("%w: only the project's lead closes it", ErrNotSteward)
	}
	if p.State != ProjectOpen {
		return ProjectView{}, fmt.Errorf("%w: the project is already closed", ErrInvalid)
	}
	if len(in.Lessons) > MaxLessons {
		return ProjectView{}, fmt.Errorf("%w: at most %d lessons", ErrInvalid, MaxLessons)
	}
	var lessons []Lesson
	for _, l := range in.Lessons {
		l, err := x.cleanCited(ctx, s, l)
		if err != nil {
			return ProjectView{}, err
		}
		lessons = append(lessons, Lesson{Text: l.Text, Cite: l.Cite})
	}
	d, err := x.def(p.Space, "close-out")
	if err != nil {
		return ProjectView{}, err
	}
	now := x.now()
	c, ok, err := x.Stewards.Agent.Collections.GetCollection(ctx, p.Tenant, p.Collection)
	if err != nil || !ok {
		return ProjectView{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	c.Closed = &now
	if err := x.Stewards.Agent.Collections.PutCollection(ctx, c); err != nil {
		return ProjectView{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	for i, l := range lessons {
		sq := StewardQuestion{ID: x.Stewards.NewID("qs_"), Tenant: p.Tenant, Asker: s.User(), Space: SpaceKey(p.Space), At: now,
			Question: firstLine("Lesson from " + p.Name + ": " + l.Text), Draft: &NoteDraft{Title: "Lesson from " + p.Name, Comment: l.Text},
			Shared: []SharedCite{{Cite: l.Cite}}, Project: &ProjectRef{Project: p.ID, Kind: "lesson", Lesson: i}}
		if err := x.Stewards.Store.PutStewardQuestion(ctx, sq); err != nil {
			return ProjectView{}, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		lessons[i].Item = sq.ID
	}
	p.State, p.Closed, p.Lessons = ProjectClosed, &now, lessons
	if err := x.projectAudit(ctx, s.User(), p, "project_closed", fmt.Sprintf("%d lessons proposed", len(lessons))); err != nil {
		return ProjectView{}, err
	}
	if err := x.putProject(ctx, p); err != nil {
		return ProjectView{}, err
	}
	if p.Kickoff != "" {
		if r, err := x.get(ctx, p.Kickoff); err == nil {
			if _, err := x.live(ctx, r); err != nil { // the closed collection stops it (LAW-29)
				return ProjectView{}, err
			}
		}
	}
	r, err := x.start(ctx, s.User(), p.Space, "close-out", d, p.Collection)
	if err != nil {
		return ProjectView{}, err
	}
	p.CloseOut = r.ID
	if err := x.putProject(ctx, p); err != nil {
		return ProjectView{}, err
	}
	return x.projectView(ctx, s, p)
}

// ProjectsIn are the projects in space s may read, newest first.
func (x Rhythms) ProjectsIn(ctx context.Context, s tenancy.Scope, space string) ([]ProjectView, error) {
	if x.Projects == nil {
		return []ProjectView{}, nil
	}
	ps, err := x.Projects.Projects(ctx, x.Stewards.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: projects: %v", ErrUnavailable, err)
	}
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].At.After(ps[j].At) })
	out := []ProjectView{}
	for _, p := range ps {
		if p.Space != space {
			continue
		}
		v, err := x.projectView(ctx, s, p)
		if errors.Is(err, ErrInvalid) {
			continue // not theirs to read
		}
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (x Rhythms) projectOf(ctx context.Context, collection string) (Project, error) {
	ps, err := x.Projects.Projects(ctx, x.Stewards.Agent.Tenant)
	if err != nil {
		return Project{}, fmt.Errorf("%w: projects: %v", ErrUnavailable, err)
	}
	for _, p := range ps {
		if p.Collection == collection {
			return p, nil
		}
	}
	return Project{}, fmt.Errorf("%w: the collection holds no project", ErrInvalid)
}

// nameWords are a name's words of four letters or more, lower case.
func nameWords(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') }) {
		if len(w) >= 4 {
			out[w] = true
		}
	}
	return out
}

// kickoffBrief is the lead's brief of up to MaxSimilarProjects closed projects whose names
// share words with this one: what each decided and the lessons its stewards published,
// each line cited, and only lines whose cite the lead can open now (LAW-01).
func (x Rhythms) kickoffBrief(ctx context.Context, s tenancy.Scope, r RhythmRun) (string, error) {
	p, err := x.projectOf(ctx, r.Subject)
	if err != nil {
		return "", err
	}
	ps, err := x.Projects.Projects(ctx, p.Tenant)
	if err != nil {
		return "", fmt.Errorf("%w: projects: %v", ErrUnavailable, err)
	}
	mine := nameWords(p.Name)
	type cand struct {
		p     Project
		score int
	}
	var cs []cand
	for _, o := range ps {
		if o.ID == p.ID || o.State != ProjectClosed {
			continue
		}
		n := 0
		for w := range nameWords(o.Name) {
			if mine[w] {
				n++
			}
		}
		if n > 0 {
			cs = append(cs, cand{o, n})
		}
	}
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].score != cs[j].score {
			return cs[i].score > cs[j].score
		}
		return cs[i].p.At.After(cs[j].p.At)
	})
	var b strings.Builder
	shown := 0
	for _, c := range cs {
		if shown == MaxSimilarProjects {
			break
		}
		ds, _, err := x.openDecisions(ctx, s, c.p.Decisions)
		if err != nil {
			return "", err
		}
		var lines []string
		for _, d := range ds {
			lines = append(lines, fmt.Sprintf("  - Decided: %s [%s]", d.Text, d.Cite))
		}
		ls, err := x.publishedLessons(ctx, s, c.p)
		if err != nil {
			return "", err
		}
		for _, l := range ls {
			lines = append(lines, fmt.Sprintf("  - Lesson: %s [%s]", l.Text, l.Cite))
		}
		if len(lines) == 0 {
			continue // nothing of it the lead can open: not even its name is shown
		}
		owner := "its lead"
		if es, err := x.Stewards.Agent.emailsOf(ctx, c.p.Lead); err == nil && len(es) > 0 {
			owner = es[0]
		}
		fmt.Fprintf(&b, "- %s, led by %s\n%s\n", c.p.Name, owner, strings.Join(lines, "\n"))
		shown++
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// publishedLessons are a project's lessons a steward published as notes, whose cite s can
// open now.
func (x Rhythms) publishedLessons(ctx context.Context, s tenancy.Scope, p Project) ([]Lesson, error) {
	var cites []string
	for _, l := range p.Lessons {
		cites = append(cites, l.Cite+"@0")
	}
	open, err := x.Stewards.Agent.opensCites(ctx, s, cites)
	if err != nil {
		return nil, err
	}
	var out []Lesson
	for _, l := range p.Lessons {
		if !open[l.Cite+"@0"] || l.Item == "" {
			continue
		}
		q, found, err := x.Stewards.Store.GetStewardQuestion(ctx, p.Tenant, l.Item)
		if err != nil {
			return nil, fmt.Errorf("%w: questions: %v", ErrUnavailable, err)
		}
		if found && q.Reply != nil && q.Reply.Note != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

// retrospective is the lead's close-out draft: the collection's goals and their status, the
// decision log, the waits still open on its documents and the lessons proposed, each line
// cited and checked as the lead (SB-4804).
func (x Rhythms) retrospective(ctx context.Context, s tenancy.Scope, r RhythmRun) (string, error) {
	p, err := x.projectOf(ctx, r.Subject)
	if err != nil {
		return "", err
	}
	c, ok, err := x.Stewards.Agent.Collections.GetCollection(ctx, p.Tenant, p.Collection)
	if err != nil || !ok {
		return "", fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	var b strings.Builder
	var docs []string
	for _, it := range c.Items {
		if !strings.HasPrefix(it, "sq_") {
			docs = append(docs, it)
		}
	}
	if lk, ok := x.Stewards.Agent.Index.(ports.DocLookup); ok && len(docs) > 0 {
		titles, _, err := x.Stewards.Agent.docTitles(ctx, s, docs)
		if err != nil {
			return "", err
		}
		var refs []ports.SourceRef
		for _, d := range docs {
			if kind, id, ok := strings.Cut(d, ":"); ok {
				refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
			}
		}
		chunks, err := lk.Docs(ctx, p.Tenant, ports.DocQuery{Refs: refs})
		if err != nil {
			return "", fmt.Errorf("%w: index: %v", ErrUnavailable, err)
		}
		for _, ch := range latest(chunks) {
			doc := ch.Kind + ":" + ch.SourceID
			t, open := titles[doc]
			g, isGoal := ParseGoal(ch.Text)
			if !open || !isGoal || ch.Start != 0 {
				continue
			}
			if t == "" {
				t = doc
			}
			state := "not met (" + orDash(g.Status) + ")"
			if g.Status == "done" {
				state = "met"
			}
			fmt.Fprintf(&b, "- Goal %s: %s [%s]\n", t, state, doc)
		}
	}
	ds, _, err := x.openDecisions(ctx, s, p.Decisions)
	if err != nil {
		return "", err
	}
	for _, d := range ds {
		fmt.Fprintf(&b, "- Decided: %s [%s]\n", d.Text, d.Cite)
	}
	if w, err := x.Stewards.Waits(ctx, s, p.Space); err == nil {
		for _, it := range w.Waiting {
			if contains(docs, it.ID) {
				fmt.Fprintf(&b, "- Still waiting on %s: %s [%s]\n", it.On, it.Title, it.ID)
			}
		}
	} else if !errors.Is(err, ErrNotMember) && !errors.Is(err, ErrInvalid) {
		return "", err
	}
	open, err := x.Stewards.Agent.opensCites(ctx, s, lessonCites(p.Lessons))
	if err != nil {
		return "", err
	}
	for _, l := range p.Lessons {
		if open[l.Cite+"@0"] {
			fmt.Fprintf(&b, "- Lesson proposed to the stewards: %s [%s]\n", l.Text, l.Cite)
		}
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func lessonCites(ls []Lesson) []string {
	var out []string
	for _, l := range ls {
		out = append(out, l.Cite+"@0")
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "no status"
	}
	return s
}
