package slackapp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ports"
)

// fakeSpaces is a one-tenant registry (adapters may not import a sibling's).
type fakeSpaces struct {
	all  []ports.Space
	Fail error
}

func (f *fakeSpaces) PutSpace(_ context.Context, s ports.Space) error {
	f.all = append(f.all, s)
	return nil
}
func (f *fakeSpaces) DeleteSpace(context.Context, string, string) error { return nil }
func (f *fakeSpaces) Space(_ context.Context, tenant, id string) (ports.Space, bool, error) {
	if f.Fail != nil {
		return ports.Space{}, false, f.Fail
	}
	for _, s := range f.all {
		if s.Tenant == tenant && s.ID == id {
			return s, true, nil
		}
	}
	return ports.Space{}, false, nil
}
func (f *fakeSpaces) Spaces(_ context.Context, tenant string) ([]ports.Space, error) {
	if f.Fail != nil {
		return nil, f.Fail
	}
	return f.all, nil
}

// SB-1007: a space's channel asks in that space for its members; a non-member gets no
// answer, never a company-wide one; an unmapped channel or a DM asks company-wide.
func TestChannelsAskInTheirSpace(t *testing.T) {
	st := &fakeSpaces{}
	if err := st.PutSpace(context.Background(), ports.Space{ID: "spc_finance", Tenant: tenant, Name: "finance",
		Members: []string{"group:finance"}, Channels: []string{"#finance-ask"}}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		channel string
		groups  []string
		space   string // "" means no answer
		msg     string
	}{
		{"finance-ask", []string{"finance"}, "spc_finance", ""},
		{"finance-ask", []string{"staff"}, "", MsgNotMember},
		{"general", []string{"staff"}, "company", ""},
		{"directmessage", []string{"staff"}, "company", ""},
	} {
		ag := &agent{}
		a, posted := newApp(directory, ag)
		a.Spaces = st
		groups := tc.groups
		a.Groups = func(context.Context, string, string) ([]string, error) { return groups, nil }
		form := command("UALICE", "What is the forecast?")
		form.Set("channel_name", tc.channel)
		send(a, "/chat/slack/commands", form, now, "")
		switch {
		case tc.space == "" && (len(ag.spaces) != 0 || len(*posted) != 1 || !strings.Contains((*posted)[0].Text, "space you aren't a member of")):
			t.Errorf("%s as %v: asked %v, posted %+v", tc.channel, tc.groups, ag.spaces, *posted)
		case tc.space != "" && (len(ag.spaces) != 1 || ag.spaces[0] != tc.space):
			t.Errorf("%s as %v: asked in %v, want %s", tc.channel, tc.groups, ag.spaces, tc.space)
		}
	}
	ag := &agent{}
	a, posted := newApp(directory, ag)
	st.Fail = errors.New("down")
	a.Spaces = st
	form := command("UALICE", "What is the forecast?")
	form.Set("channel_name", "finance-ask")
	send(a, "/chat/slack/commands", form, now, "")
	if len(ag.spaces) != 0 || len(*posted) != 1 || (*posted)[0].Text != MsgSpacesDown {
		t.Fatalf("a failing registry must give no answer: asked %v, posted %+v", ag.spaces, *posted)
	}
}
