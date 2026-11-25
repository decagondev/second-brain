package manifest

import (
	"fmt"
	"net/url"
)

// Events are the signed events out (sb-tenant(5), events; SB-4404): the URL to post to and
// the kinds chosen. Nil sends none.
type Events struct {
	URL   string   `json:"url"`
	Kinds []string `json:"kinds"`
}

// EventKinds are the kinds a cell can send.
var EventKinds = []string{"decision", "goal_at_risk", "request_overdue", "policy_published", "action_due"}

// events checks the events field.
func (c *checker) events(v *node) {
	f := "events"
	if !c.mapOf(v, f, `{ url: https://hooks.acme.example/brain, kinds: [decision] }`) {
		return
	}
	e := &Events{}
	for j, k := range v.keys {
		fk := f + "." + k.value
		switch k.value {
		case "url":
			s, ok := c.scalar(v.vals[j], fk)
			if !ok {
				continue
			}
			u, err := url.Parse(s)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
				c.add(v.vals[j], fk, "%q must be an https:// URL with a host", s)
				continue
			}
			e.URL = s
		case "kinds":
			vs, ns, ok := c.list(v.vals[j], fk, "[decision, goal_at_risk]")
			if !ok {
				continue
			}
			for m, kind := range vs {
				fm := fmt.Sprintf("%s[%d]", fk, m)
				switch {
				case !contains(EventKinds, kind):
					c.add(ns[m], fm, "%q is not an event kind; use %s", kind, orList(EventKinds))
				case contains(e.Kinds, kind):
					c.add(ns[m], fm, "%q is listed twice", kind)
				default:
					e.Kinds = append(e.Kinds, kind)
				}
			}
		default:
			c.add(k, fk, "unknown field in events; use url or kinds")
		}
	}
	if e.URL == "" || len(e.Kinds) == 0 {
		c.add(v, f, "events need a url and at least one kind")
		return
	}
	c.t.Events = e
}
