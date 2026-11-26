package ask_test

import (
	"context"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
)

type confirms map[string]ask.WideConfirm

func (c confirms) PutWideConfirm(_ context.Context, x ask.WideConfirm) error {
	c[x.Doc] = x
	return nil
}
func (c confirms) WideConfirm(_ context.Context, _, doc string) (ask.WideConfirm, bool, error) {
	x, ok := c[doc]
	return x, ok, nil
}

// SB-5401, SB-5402: a document open to the domain with personal data is held from alice;
// warn lets it through and counts it; a big group counts as wide only above the limit, and
// a confirmation for another version releases nothing.
func TestWideSensitiveIsHeld(t *testing.T) {
	m := &model{reply: `{"claims":[{"text":"Limit is 500.","cites":["C1"]}]}`}
	a, _ := setup(t, m)
	ix := a.Index.(index)
	ix[0].ACLPrefilter = []string{"domain:acme.example"}
	ix[0].Flags = []string{ingest.FlagPersonal}
	a.Index = ix
	cf := confirms{}
	a.Wide = &ask.Wide{Confirms: cf}
	resp, err := a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Held != 1 || len(m.got) != 0 {
		t.Fatalf("held %d, model calls %d", resp.Held, len(m.got))
	}
	cf["docs:refunds"] = ask.WideConfirm{Doc: "docs:refunds", Version: 1, ACL: ask.ACLHash(ix[0].ACLPrefilter)}
	if resp, _ = a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"}); resp.Held != 1 {
		t.Fatalf("an older version's confirmation released it: %+v", resp)
	}
	cf["docs:refunds"] = ask.WideConfirm{Doc: "docs:refunds", Version: 2, ACL: ask.ACLHash(ix[0].ACLPrefilter)}
	if resp, _ = a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"}); resp.Held != 0 || len(m.got) != 1 {
		t.Fatalf("confirmed but held: %+v", resp)
	}
	delete(cf, "docs:refunds")
	ix[0].Spaces = []string{"spc_finance"} // asked company-wide, warn only when every holder warns
	a.Index = ix
	a.Wide.Mode = func(string) string { return ask.WideWarn }
	if resp, _ = a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"}); resp.Held != 0 || resp.WideWarned != 1 {
		t.Fatalf("warn: %+v", resp)
	}

	a.Wide.Mode = nil
	ix[0].ACLPrefilter = []string{"group:all-staff"}
	a.Index = ix
	a.Wide.GroupSize = func(_ context.Context, g string) (int, bool) { return 900, g == "all-staff" }
	if resp, _ = a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"}); resp.Held != 1 {
		t.Fatalf("a group of 900: %+v", resp)
	}
	a.Wide.Limit = 1000
	if resp, _ = a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"}); resp.Held != 0 {
		t.Fatalf("a group under the limit: %+v", resp)
	}
	ix[0].ACLPrefilter, ix[0].Flags = []string{"anyone"}, nil
	a.Index = ix
	if resp, _ = a.Ask(context.Background(), scope(t, "alice"), ask.Request{Question: "refund limit?"}); resp.Held != 0 {
		t.Fatalf("wide with nothing sensitive: %+v", resp)
	}
}
