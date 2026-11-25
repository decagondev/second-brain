package ask

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/tenancy"
)

// Collections (SB-4304): the documents and saved questions for one piece of work, by id
// only, shared with named people. Each reader sees only the documents they can open now;
// the rest are counted, never named (LAW-01).

// Collection limits.
const (
	MaxCollectionItems  = 50
	MaxCollectionName   = 120
	MaxCollectionShared = 20
	// CollectionStaleDays is how long a collection goes unasked before its owner is
	// offered to remove it.
	CollectionStaleDays = 90
)

// Collection is one collection as kept: ids only.
type Collection struct {
	ID     string    `json:"id"`
	Tenant string    `json:"tenant"`
	Owner  string    `json:"owner"` // the owner's user id
	Name   string    `json:"name"`
	Items  []string  `json:"items"`            // KIND:ID or sq_…
	Shared []string  `json:"shared,omitempty"` // emails, lower case
	At     time.Time `json:"at"`
	Used   time.Time `json:"used"`
}

// CollectionStore keeps collections.
type CollectionStore interface {
	PutCollection(ctx context.Context, c Collection) error
	GetCollection(ctx context.Context, tenant, id string) (Collection, bool, error)
	Collections(ctx context.Context, tenant string) ([]Collection, error)
	DeleteCollection(ctx context.Context, tenant, id string) error
}

// CollectionItem is one item a reader may see: a document's cite and title, or a saved
// question's text.
type CollectionItem struct {
	Item     string `json:"item"`
	Title    string `json:"title,omitempty"`
	Question string `json:"question,omitempty"`
}

// CollectionView is a collection as one reader sees it.
type CollectionView struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Owner  string           `json:"owner"`
	Mine   bool             `json:"mine"`
	Items  []CollectionItem `json:"items"`
	Hidden int              `json:"hidden"`
	Shared []string         `json:"shared,omitempty"`
	At     time.Time        `json:"at"`
	Stale  bool             `json:"stale,omitempty"`
}

// CollectionInput is a new collection, or a collection's new name, items and people.
type CollectionInput struct {
	Name   string   `json:"name"`
	Items  []string `json:"items"`
	Shared []string `json:"shared"`
}

// CollectionService is the collections service for one cell.
type CollectionService struct {
	Agent Agent
	Store CollectionStore
	Saved QuestionStore
	Audit Auditor
	// Known is whether an email is one active user now (LAW-12); nil knows nobody.
	Known func(ctx context.Context, email string) bool
	NewID func(prefix string) string
	Now   func() time.Time
}

func (x CollectionService) now() time.Time { return x.Now().UTC() }

// emailsOf are a person's addresses in the directory now, lower case.
func (a Agent) emailsOf(ctx context.Context, user string) ([]string, error) {
	if a.Emails == nil {
		return nil, nil
	}
	es, err := a.Emails(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("%w: directory: %v", ErrUnavailable, err)
	}
	for i := range es {
		es[i] = strings.ToLower(strings.TrimSpace(es[i]))
	}
	return es, nil
}

// readable is whether s's person owns c or it is shared with one of their addresses.
func readable(c Collection, user string, emails []string) bool {
	if c.Owner == user {
		return true
	}
	for _, e := range emails {
		for _, sh := range c.Shared {
			if e != "" && e == sh {
				return true
			}
		}
	}
	return false
}

// readCollection is a collection s may read; anything else is ErrInvalid.
func (a Agent) readCollection(ctx context.Context, s tenancy.Scope, store CollectionStore, id string) (Collection, error) {
	if store == nil {
		return Collection{}, fmt.Errorf("%w: this cell keeps no collections", ErrInvalid)
	}
	c, found, err := store.GetCollection(ctx, a.Tenant, strings.TrimSpace(id))
	if err != nil {
		return Collection{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	emails, err := a.emailsOf(ctx, s.User())
	if err != nil {
		return Collection{}, err
	}
	if !found || c.Tenant != a.Tenant || !readable(c, s.User(), emails) {
		return Collection{}, fmt.Errorf("%w: no such collection", ErrInvalid)
	}
	return c, nil
}

// collectionOr is docs, or with none, the documents of collection id when there is one.
func (a Agent) collectionOr(ctx context.Context, s tenancy.Scope, id string, docs []string) ([]string, error) {
	if d := onlyDocs(ctx); len(d) > 0 && len(docs) == 0 {
		return d, nil // a catch-up's changed documents (SB-4506)
	}
	if id == "" || len(docs) > 0 {
		return docs, nil
	}
	return a.collectionDocs(ctx, s, id)
}

// collectionDocs are the documents of a collection s may read, for an ask; asking marks it
// used.
func (a Agent) collectionDocs(ctx context.Context, s tenancy.Scope, id string) ([]string, error) {
	c, err := a.readCollection(ctx, s, a.Collections, id)
	if err != nil {
		return nil, err
	}
	var docs []string
	for _, it := range c.Items {
		if !strings.HasPrefix(it, "sq_") {
			docs = append(docs, it)
		}
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("%w: the collection holds no documents", ErrInvalid)
	}
	c.Used = a.Now().UTC()
	if err := a.Collections.PutCollection(ctx, c); err != nil {
		return nil, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	return docs, nil
}

// clean checks an input: a name, at most MaxCollectionItems items, each KIND:ID (its
// version dropped) or one of the owner's saved questions, and active people to share with.
func (x CollectionService) clean(ctx context.Context, s tenancy.Scope, in CollectionInput) (Collection, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > MaxCollectionName || strings.ContainsAny(name, "\n\r") {
		return Collection{}, fmt.Errorf("%w: a collection's name is 1 to %d characters on one line", ErrInvalid, MaxCollectionName)
	}
	if len(in.Items) > MaxCollectionItems {
		return Collection{}, fmt.Errorf("%w: at most %d items", ErrInvalid, MaxCollectionItems)
	}
	if len(in.Shared) > MaxCollectionShared {
		return Collection{}, fmt.Errorf("%w: share with at most %d people", ErrInvalid, MaxCollectionShared)
	}
	saved := map[string]bool{}
	if x.Saved != nil {
		qs, err := x.Saved.SavedOf(ctx, x.Agent.Tenant, s.User())
		if err != nil {
			return Collection{}, fmt.Errorf("%w: saved questions: %v", ErrUnavailable, err)
		}
		for _, q := range qs {
			saved[q.ID] = true
		}
	}
	c := Collection{Tenant: x.Agent.Tenant, Owner: s.User(), Name: name, Items: []string{}}
	seen := map[string]bool{}
	for _, it := range in.Items {
		it = strings.TrimSpace(it)
		if strings.HasPrefix(it, "sq_") {
			if !saved[it] {
				return Collection{}, fmt.Errorf("%w: %s isn't one of your saved questions", ErrInvalid, it)
			}
		} else {
			ks, err := documentKeys([]string{it})
			if err != nil {
				return Collection{}, err
			}
			it = ks[0]
		}
		if !seen[it] {
			seen[it] = true
			c.Items = append(c.Items, it)
		}
	}
	for _, e := range in.Shared {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" || x.Known == nil || !x.Known(ctx, e) {
			return Collection{}, fmt.Errorf("%w: share only with active people", ErrInvalid)
		}
		if !contains(c.Shared, e) {
			c.Shared = append(c.Shared, e)
		}
	}
	return c, nil
}

func (x CollectionService) audit(ctx context.Context, s tenancy.Scope, c Collection, state string) error {
	items := append([]string(nil), c.Items...)
	sort.Strings(items)
	if _, err := x.Audit.Write(ctx, audit.Event{Actor: s.User(), Kind: "access", Subject: c.ID, State: state,
		PayloadSHA256: hashOf(strings.Join(items, "\n")), Reason: "collection"}); err != nil {
		return fmt.Errorf("%w: audit: %v", ErrUnavailable, err)
	}
	return nil
}

func (x CollectionService) ready(s tenancy.Scope) error {
	if err := s.Check(); err != nil || s.Tenant() != x.Agent.Tenant {
		return fmt.Errorf("%w: no valid scope for this cell", ErrInvalid)
	}
	if x.Store == nil {
		return fmt.Errorf("%w: this cell keeps no collections", ErrUnavailable)
	}
	return nil
}

// Create makes a collection owned by s's person.
func (x CollectionService) Create(ctx context.Context, s tenancy.Scope, in CollectionInput) (CollectionView, error) {
	if err := x.ready(s); err != nil {
		return CollectionView{}, err
	}
	c, err := x.clean(ctx, s, in)
	if err != nil {
		return CollectionView{}, err
	}
	c.ID, c.At = x.NewID("col_"), x.now()
	c.Used = c.At
	if err := x.audit(ctx, s, c, "collected"); err != nil {
		return CollectionView{}, err
	}
	if err := x.Store.PutCollection(ctx, c); err != nil {
		return CollectionView{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	return x.view(ctx, s, c, nil)
}

// own is one of s's own collections; anyone else's is ErrNotFound (EPERM).
func (x CollectionService) own(ctx context.Context, s tenancy.Scope, id string) (Collection, error) {
	c, found, err := x.Store.GetCollection(ctx, x.Agent.Tenant, strings.TrimSpace(id))
	if err != nil {
		return Collection{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	if !found || c.Tenant != x.Agent.Tenant || c.Owner != s.User() {
		return Collection{}, ErrNotFound
	}
	return c, nil
}

// Update replaces the name, items and people of one of s's own collections.
func (x CollectionService) Update(ctx context.Context, s tenancy.Scope, id string, in CollectionInput) (CollectionView, error) {
	if err := x.ready(s); err != nil {
		return CollectionView{}, err
	}
	old, err := x.own(ctx, s, id)
	if err != nil {
		return CollectionView{}, err
	}
	c, err := x.clean(ctx, s, in)
	if err != nil {
		return CollectionView{}, err
	}
	c.ID, c.At, c.Used = old.ID, old.At, x.now()
	if err := x.audit(ctx, s, c, "changed"); err != nil {
		return CollectionView{}, err
	}
	if err := x.Store.PutCollection(ctx, c); err != nil {
		return CollectionView{}, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	return x.view(ctx, s, c, nil)
}

// Delete removes one of s's own collections.
func (x CollectionService) Delete(ctx context.Context, s tenancy.Scope, id string) error {
	if err := x.ready(s); err != nil {
		return err
	}
	c, err := x.own(ctx, s, id)
	if err != nil {
		return err
	}
	if err := x.audit(ctx, s, c, "removed"); err != nil {
		return err
	}
	if err := x.Store.DeleteCollection(ctx, x.Agent.Tenant, c.ID); err != nil {
		return fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	return nil
}

// List is every collection s's person owns or that is shared with them, newest first, each
// as they may see it now.
func (x CollectionService) List(ctx context.Context, s tenancy.Scope) ([]CollectionView, error) {
	if err := x.ready(s); err != nil {
		return nil, err
	}
	all, err := x.Store.Collections(ctx, x.Agent.Tenant)
	if err != nil {
		return nil, fmt.Errorf("%w: collections: %v", ErrUnavailable, err)
	}
	emails, err := x.Agent.emailsOf(ctx, s.User())
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].At.After(all[j].At) })
	out := []CollectionView{}
	saved := map[string]map[string]string{} // owner -> saved question id -> text
	for _, c := range all {
		if c.Tenant != x.Agent.Tenant || !readable(c, s.User(), emails) {
			continue
		}
		v, err := x.view(ctx, s, c, saved)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// view is c as s's person may see it now: documents checked live as them, the rest counted.
func (x CollectionService) view(ctx context.Context, s tenancy.Scope, c Collection, saved map[string]map[string]string) (CollectionView, error) {
	v := CollectionView{ID: c.ID, Name: c.Name, Mine: c.Owner == s.User(), Items: []CollectionItem{}, At: c.At}
	if es, err := x.Agent.emailsOf(ctx, c.Owner); err == nil && len(es) > 0 {
		v.Owner = es[0]
	}
	if v.Mine {
		v.Shared = c.Shared
		v.Stale = x.now().Sub(c.Used) > CollectionStaleDays*24*time.Hour
	}
	if saved == nil {
		saved = map[string]map[string]string{}
	}
	var docs []string
	for _, it := range c.Items {
		if !strings.HasPrefix(it, "sq_") {
			docs = append(docs, it)
		}
	}
	titles, hidden, err := x.Agent.docTitles(ctx, s, docs)
	if err != nil {
		return CollectionView{}, err
	}
	v.Hidden = hidden
	for _, it := range c.Items {
		if !strings.HasPrefix(it, "sq_") {
			if t, ok := titles[it]; ok {
				v.Items = append(v.Items, CollectionItem{Item: it, Title: t})
			}
			continue
		}
		qs, ok := saved[c.Owner]
		if !ok && x.Saved != nil {
			qs = map[string]string{}
			all, err := x.Saved.SavedOf(ctx, x.Agent.Tenant, c.Owner)
			if err != nil {
				return CollectionView{}, fmt.Errorf("%w: saved questions: %v", ErrUnavailable, err)
			}
			for _, q := range all {
				qs[q.ID] = q.Question
			}
			saved[c.Owner] = qs
		}
		if q, ok := qs[it]; ok { // one its owner removed is gone
			v.Items = append(v.Items, CollectionItem{Item: it, Question: q})
		}
	}
	return v, nil
}

// docTitles are the titles of the documents (KIND:ID) s may open now, each checked live at
// its newest version, and how many of the others there are.
func (a Agent) docTitles(ctx context.Context, s tenancy.Scope, docs []string) (map[string]string, int, error) {
	out := map[string]string{}
	if len(docs) == 0 {
		return out, 0, nil
	}
	cites, err := a.newestCites(ctx, docs)
	if err != nil {
		return nil, 0, err
	}
	var list []string
	for _, d := range docs {
		if c, ok := cites[d]; ok {
			list = append(list, c)
		}
	}
	rs, _, err := a.CheckCites(ctx, s, list)
	if err != nil {
		return nil, 0, err
	}
	for _, r := range rs {
		doc, _, _ := strings.Cut(r.Cite, "@")
		out[doc] = r.Title
	}
	return out, len(docs) - len(out), nil
}

// newestCites is the newest indexed version of each document (KIND:ID), as a cite.
func (a Agent) newestCites(ctx context.Context, docs []string) (map[string]string, error) {
	lk, ok := a.Index.(ports.DocLookup)
	if !ok {
		return nil, fmt.Errorf("%w: the index can't look up documents", ErrUnavailable)
	}
	var refs []ports.SourceRef
	for _, d := range docs {
		kind, id, _ := strings.Cut(d, ":")
		refs = append(refs, ports.SourceRef{Kind: kind, ID: id})
	}
	chunks, err := lk.Docs(ctx, a.Tenant, ports.DocQuery{Refs: refs})
	if err != nil {
		return nil, fmt.Errorf("%w: index: %v", ErrUnavailable, err)
	}
	out := map[string]string{}
	for doc, c := range firsts(latest(chunks)) {
		out[doc] = c.Cite
	}
	return out, nil
}
