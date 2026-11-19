package verify

import (
	"context"
	"strings"
	"testing"
)

type fake struct {
	answer  Answer
	foreign bool // answers cite the foreign id
	allowed bool
}

func (f fake) Users(context.Context) ([]string, error)     { return []string{"u1", "u2"}, nil }
func (f fake) Questions(context.Context) ([]string, error) { return []string{"q"}, nil }
func (f fake) Ask(_ context.Context, _, q string) (Answer, error) {
	a := f.answer
	if f.foreign && strings.Contains(q, "docs:other") {
		a.Cites = append(a.Cites, Cite{Doc: "docs:other", Connector: "docs"})
	}
	return a, nil
}
func (f fake) Allowed(context.Context, string, string) (bool, error) { return f.allowed, nil }
func (f fake) ForeignIDs(context.Context) ([]string, error)          { return []string{"docs:other"}, nil }
func (f fake) Timeout(string) (func(), error)                        { return func() {}, nil }
func (f fake) Connectors() []string                                  { return []string{"docs"} }

func good() Answer {
	return Answer{Cites: []Cite{{Doc: "docs:a", Connector: "chat"}}, Cited: 1, Uncited: 1, LiveChecked: map[string]bool{"docs:a": true}}
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	if r := Run(ctx, fake{answer: good(), allowed: true}, Options{Sample: 5}); !r.Passed() {
		t.Fatalf("%s", r)
	}
	cases := map[string]fake{
		"not allowed now":    {answer: good(), allowed: false},
		"cross tenant":       {answer: good(), allowed: true, foreign: true},
		"unlabelled claim":   {answer: Answer{Cites: good().Cites, Cited: 1, Unlabelled: 1, LiveChecked: good().LiveChecked}, allowed: true},
		"no live check seen": {answer: Answer{Cites: good().Cites, Cited: 1, LiveChecked: map[string]bool{}}, allowed: true},
	}
	for name, f := range cases {
		if r := Run(ctx, f, Options{Sample: 5}); r.Passed() {
			t.Errorf("%s passed:\n%s", name, r)
		}
	}
	if r := Run(ctx, fake{answer: good(), allowed: true}, Options{Laws: []string{"LAW-99"}}); r.Passed() || r.Results[0].Status != "unknown" {
		t.Errorf("an unknown check must fail: %s", r)
	}
}
