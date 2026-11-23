package ask_test

import (
	"context"
	"strings"
	"testing"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/ingest"
)

// SB-1904: a flagged chunk's snippet shows only the number's last four characters.
func TestSearchSnippetMasksPersonalData(t *testing.T) {
	a, _ := setup(t, &model{})
	ix := a.Index.(index)
	ix[0].Text = "Pay supplier refunds to DE89 3704 0044 0532 0130 00 within 30 days."
	ix[0].Flags = []string{ingest.FlagPersonal}
	a.Index = ix
	res, err := a.Search(context.Background(), scope(t, "alice"), ask.SearchRequest{Query: "refund"})
	if err != nil || len(res.Results) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	sn := res.Results[0].Snippet
	if strings.Contains(sn, "3704") || !strings.Contains(sn, "30 00") || !strings.Contains(sn, "within 30 days") {
		t.Fatalf("snippet %q", sn)
	}
}
