package laws

import (
	"context"
	"fmt"
	"math/rand"

	"example.com/second-brain/internal/adapters/connectors/hubspot"
	"example.com/second-brain/internal/adapters/connectors/hubspot/hubspottest"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/obs"
)

// LAW-01 on CRM records (SB-610): the live verdict on a company or deal follows HubSpot's
// record-level permissions at this moment (the user's role level, teams, the record's
// owner and team) through reassignments, role and team changes, deactivation and
// archiving. A level the tenant hasn't mapped is Unknown, never Allow.

// CRMWorld is a HubSpot portal and the connector over it. Levels are "all", "team",
// "owned", "admin" (super admin) and "none" (a role missing from the role map).
type CRMWorld interface {
	AddUser(email, level string, teams ...string)
	SetLevel(email, level string)
	SetTeams(email string, teams ...string)
	Deactivate(email string)
	Put(id, owner, team string) // id is companies/N or deals/N; owner is an email
	Reassign(id, owner, team string)
	Archive(id string)
	Live(email, id string) (string, error)
}

// NewCRMWorld builds the system under test: the HubSpot connector and its fake.
var NewCRMWorld = func() CRMWorld {
	f := hubspottest.New()
	c := &hubspot.Connector{Base: f.Base(), HTTP: f.Server.Client(), Redact: obs.NewRedactor(),
		Creds: func(context.Context) (string, error) {
			return `{"token": "` + f.Token + `", "roles": {"r_all": "all", "r_team": "team", "r_owned": "owned"}}`, nil
		}}
	return &crmWorld{f: f, c: c}
}

type crmWorld struct {
	f *hubspottest.Fake
	c *hubspot.Connector
	n int
}

func crmRole(level string) (string, bool) {
	if level == "admin" {
		return "r_none", true
	}
	return "r_" + level, false
}

func (w *crmWorld) AddUser(email, level string, teams ...string) {
	w.n++
	role, admin := crmRole(level)
	w.f.AddUser(fmt.Sprintf("U%d", w.n), email, role, admin, teams...)
}
func (w *crmWorld) SetLevel(email, level string) {
	role, admin := crmRole(level)
	w.f.SetRole(email, role, admin)
}
func (w *crmWorld) SetTeams(email string, teams ...string) { w.f.SetTeams(email, teams...) }
func (w *crmWorld) Deactivate(email string)                { w.f.Deactivate(email) }
func (w *crmWorld) Put(id, owner, team string) {
	w.f.Put(id, "Record "+id, "x", w.f.Owner(owner), team)
}
func (w *crmWorld) Reassign(id, owner, team string) { w.f.Reassign(id, w.f.Owner(owner), team) }
func (w *crmWorld) Archive(id string)               { w.f.Archive(id) }
func (w *crmWorld) Live(email, id string) (string, error) {
	vs, err := w.c.ACL(context.Background(), ports.UserRef{Email: email}, []ports.SourceRef{{Kind: "crm", ID: id}})
	if err != nil || len(vs) != 1 {
		return "unknown", err
	}
	return vs[0].String(), nil
}

// crmModel is the reference: HubSpot's permissions as sb-connector(3) states them.
type crmModel struct {
	level  map[string]string
	teams  map[string]map[string]bool
	gone   map[string]bool
	owner  map[string]string
	team   map[string]string
	archiv map[string]bool
}

func (m crmModel) want(email, id string) string {
	if m.gone[email] {
		return "deny"
	}
	if m.archiv[id] || m.owner[id] == "" {
		return "unknown"
	}
	mine := m.owner[id] == email
	switch m.level[email] {
	case "all", "admin":
		return "allow"
	case "team":
		if mine || m.teams[email][m.team[id]] {
			return "allow"
		}
	case "owned":
		if mine {
			return "allow"
		}
	default:
		return "unknown"
	}
	return "deny"
}

func law01CRM(r *rand.Rand, w CRMWorld) error {
	users := []string{"ann@acme.example", "bob@acme.example", "cy@acme.example", "dee@acme.example"}
	levels := []string{"all", "team", "owned", "admin", "none"}
	teamNames := []string{"T1", "T2", "T3"}
	m := crmModel{level: map[string]string{}, teams: map[string]map[string]bool{}, gone: map[string]bool{},
		owner: map[string]string{}, team: map[string]string{}, archiv: map[string]bool{}}
	pickTeams := func() []string {
		var out []string
		for _, t := range teamNames {
			if r.Intn(2) == 0 {
				out = append(out, t)
			}
		}
		return out
	}
	setTeams := func(u string, ts []string) {
		m.teams[u] = map[string]bool{}
		for _, t := range ts {
			m.teams[u][t] = true
		}
	}
	for _, u := range users {
		l, ts := levels[r.Intn(len(levels))], pickTeams()
		w.AddUser(u, l, ts...)
		m.level[u] = l
		setTeams(u, ts)
	}
	var ids []string
	for i := 0; i < 2+r.Intn(4); i++ {
		id := fmt.Sprintf("%s/%d", []string{"companies", "deals"}[r.Intn(2)], i)
		o, t := users[r.Intn(len(users))], teamNames[r.Intn(len(teamNames))]
		w.Put(id, o, t)
		m.owner[id], m.team[id] = o, t
		ids = append(ids, id)
	}
	check := func(step string) error {
		for _, u := range users {
			for _, id := range append(ids, "deals/missing") {
				got, err := w.Live(u, id)
				if err != nil {
					return err
				}
				want := m.want(u, id)
				if got != want {
					return fmt.Errorf("after %s, %s on %s is %s; HubSpot's permissions say %s (level %s, teams %v, owner %s, team %s)",
						step, u, id, got, want, m.level[u], m.teams[u], m.owner[id], m.team[id])
				}
			}
		}
		return nil
	}
	if err := check("setup"); err != nil {
		return err
	}
	for round := 0; round < 4; round++ {
		u, id := users[r.Intn(len(users))], ids[r.Intn(len(ids))]
		var step string
		switch r.Intn(5) {
		case 0:
			o, t := users[r.Intn(len(users))], teamNames[r.Intn(len(teamNames))]
			if !m.archiv[id] {
				w.Reassign(id, o, t)
				m.owner[id], m.team[id] = o, t
			}
			step = "reassigning " + id
		case 1:
			l := levels[r.Intn(len(levels))]
			w.SetLevel(u, l)
			m.level[u] = l
			step = "changing " + u + "'s role"
		case 2:
			ts := pickTeams()
			w.SetTeams(u, ts...)
			setTeams(u, ts)
			step = "changing " + u + "'s teams"
		case 3:
			w.Deactivate(u)
			m.gone[u] = true
			step = "deactivating " + u
		default:
			if !m.archiv[id] {
				w.Archive(id)
				m.archiv[id] = true
			}
			step = "archiving " + id
		}
		if err := check(step); err != nil {
			return err
		}
	}
	return nil
}

// NewHubSpotWorld is the connector-level world (LAW-03) over the HubSpot connector. Access
// means the asker (role "owned") owns the record; "script" archives the first record.
var NewHubSpotWorld = func() ConnectorWorld {
	w := NewCRMWorld().(*crmWorld)
	w.AddUser("ann@acme.example", "owned", "T1")
	w.AddUser("bob@acme.example", "owned", "T1")
	return &hubspotWorld{w: w}
}

type hubspotWorld struct {
	w *crmWorld
	n int
}

func (h *hubspotWorld) ACL(access []bool, fault string) ([]string, error) {
	h.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		owner := "bob@acme.example"
		if ok {
			owner = "ann@acme.example"
		}
		id := fmt.Sprintf("deals/%d_%d", h.n, i)
		h.w.Put(id, owner, "T1")
		refs[i] = ports.SourceRef{Kind: "crm", ID: id}
	}
	switch fault {
	case "script":
		h.w.Archive(refs[0].ID)
	case "404": // every record read fails: one fault per ref
		for range refs {
			h.w.f.FailNext("404")
		}
	case "":
	default:
		h.w.f.FailNext(fault)
	}
	vs, err := h.w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (h *hubspotWorld) Act(string, bool) (string, error) { return "", fmt.Errorf("ENOTSUP") }
func (h *hubspotWorld) Effects() int                     { return 0 }
