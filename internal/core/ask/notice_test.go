package ask

import (
	"context"
	"strings"
	"testing"
	"time"
)

type noticeMem struct {
	n  CellNotice
	ok bool
}

func (m *noticeMem) PutNotice(_ context.Context, n CellNotice) error { m.n, m.ok = n, true; return nil }
func (m *noticeMem) Notice(context.Context, string) (CellNotice, bool, error) {
	return m.n, m.ok, nil
}
func (m *noticeMem) DeleteNotice(context.Context, string) error { m.ok = false; return nil }

func TestCheckNotice(t *testing.T) {
	now := time.Date(2027, 3, 10, 9, 0, 0, 0, time.UTC)
	ok := now.Add(48 * time.Hour)
	if err := CheckNotice("The finance space is read-only until Friday.", ok, now); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		text  string
		until time.Time
	}{
		"empty":      {"  ", ok},
		"too long":   {strings.Repeat("a", MaxNoticeText+1), ok},
		"link":       {"see https://example.com", ok},
		"markup":     {"<b>bold</b>", ok},
		"brackets":   {"[here](x)", ok},
		"backtick":   {"run `x`", ok},
		"two lines":  {"one\ntwo", ok},
		"control":    {"bell\a", ok},
		"past":       {"fine", now.Add(-time.Minute)},
		"now":        {"fine", now},
		"over 7 day": {"fine", now.Add(MaxNoticeFor + time.Second)},
	} {
		if CheckNotice(c.text, c.until, now) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if CheckNotice(strings.Repeat("é", MaxNoticeText), now.Add(MaxNoticeFor), now) != nil {
		t.Error("200 characters, counted as characters, ending in exactly 7 days, is a notice")
	}
}

func TestNoticeEndsByItself(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 9, 0, 0, 0, time.UTC)
	m := &noticeMem{}
	if got := NoticeLine(ctx, m, "acme-logistics", now); got != "" {
		t.Fatalf("no notice: %q", got)
	}
	_ = m.PutNotice(ctx, CellNotice{Tenant: "acme-logistics", Text: "Upgrade tonight at 18:00.", Until: now.Add(time.Hour)})
	if got := NoticeLine(ctx, m, "acme-logistics", now); got != "Notice: Upgrade tonight at 18:00." {
		t.Fatalf("standing: %q", got)
	}
	if _, ok, _ := Standing(ctx, m, "acme-logistics", now.Add(time.Hour)); ok {
		t.Fatal("a notice at its end time no longer stands")
	}
	if _, ok, err := Standing(ctx, nil, "acme-logistics", now); ok || err != nil {
		t.Fatal("no store is no notice")
	}
}
