package ask

import (
	"context"
	"sort"

	"example.com/second-brain/internal/core/answer"
	"example.com/second-brain/internal/core/ports"
)

// MaxPeople is the most owners an answer suggests asking (SB-1702).
const MaxPeople = 3

// PersonRef is one person an answer suggests asking: an owner of its cited documents, and
// how many of its cites name their documents.
type PersonRef struct {
	Email string `json:"email"`
	Title string `json:"title,omitempty"`
	Cites int    `json:"cites"`
	// Caretaker marks someone named for documents whose owner is gone (SB-3405).
	Caretaker bool `json:"caretaker,omitempty"`
}

// owners looks owners up once per request. A lookup that fails, or finds no single active
// user, shows no owner: an owner is never guessed (sb-ask(3), Who to ask).
type owners struct {
	find func(ctx context.Context, email string) (answer.Person, bool)
	care func(ctx context.Context, doc string) string
	seen map[string]*answer.Person
	docs map[string]*answer.Person
}

func (a Agent) owners() *owners {
	return &owners{find: a.Owner, care: a.Caretaker, seen: map[string]*answer.Person{}, docs: map[string]*answer.Person{}}
}

// of is the chunk's owner when they are one active user; otherwise, for an owner who is
// none, the document's caretaker when they are one (SB-3405).
func (o *owners) of(ctx context.Context, c ports.Chunk) *answer.Person {
	if o == nil || o.find == nil || c.Owner == "" {
		return nil
	}
	p, ok := o.seen[c.Owner]
	if !ok {
		if q, found := o.find(ctx, c.Owner); found && q.Email != "" {
			p = &q
		}
		o.seen[c.Owner] = p
	}
	if p != nil || o.care == nil {
		return p
	}
	k := c.Kind + ":" + c.SourceID
	if d, ok := o.docs[k]; ok {
		return d
	}
	var out *answer.Person
	if e := o.care(ctx, k); e != "" {
		if q, found := o.find(ctx, e); found && q.Email != "" {
			q.Caretaker = true
			out = &q
		}
	}
	o.docs[k] = out
	return out
}

// people is who to ask: the owners on an answer's cites, most cited first, then by email.
func people(claims []answer.Claim) []PersonRef {
	count := map[string]*PersonRef{}
	for _, c := range claims {
		if c.Uncited {
			continue
		}
		for _, ci := range c.Cites {
			if ci.Owner == nil {
				continue
			}
			p, ok := count[ci.Owner.Email]
			if !ok {
				p = &PersonRef{Email: ci.Owner.Email, Title: ci.Owner.Title, Caretaker: ci.Owner.Caretaker}
				count[ci.Owner.Email] = p
			}
			p.Cites++
		}
	}
	out := make([]PersonRef, 0, len(count))
	for _, p := range count {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cites != out[j].Cites {
			return out[i].Cites > out[j].Cites
		}
		return out[i].Email < out[j].Email
	})
	if len(out) > MaxPeople {
		out = out[:MaxPeople]
	}
	return out
}

// ActiveOwner is the owner lookup over a directory (sb-identity(7), Owners shown to
// askers): exactly one active user with the email, or nobody.
func ActiveOwner(find func(ctx context.Context, email string) ([]ports.User, error)) func(ctx context.Context, email string) (answer.Person, bool) {
	return func(ctx context.Context, email string) (answer.Person, bool) {
		us, err := find(ctx, email)
		if err != nil {
			return answer.Person{}, false
		}
		var hit []ports.User
		for _, u := range us {
			if u.Active {
				hit = append(hit, u)
			}
		}
		if len(hit) != 1 {
			return answer.Person{}, false
		}
		return answer.Person{Email: email, Title: hit[0].Title}, true
	}
}
