package ask

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// The notice to everyone (sb-admin(3), PUT /v1/admin/notice; SB-3207): one line of plain
// text an admin sets for the whole cell, which ends by itself.

// Bounds.
const (
	MaxNoticeText = 200
	MaxNoticeFor  = 7 * 24 * time.Hour
)

// CellNotice is the cell's one notice.
type CellNotice struct {
	Tenant string    `json:"tenant"`
	Text   string    `json:"text"`
	Until  time.Time `json:"until"`
	By     string    `json:"by"`
	At     time.Time `json:"at"`
}

// NoticeStore keeps the one notice of a tenant.
type NoticeStore interface {
	PutNotice(ctx context.Context, n CellNotice) error
	Notice(ctx context.Context, tenant string) (CellNotice, bool, error)
	DeleteNotice(ctx context.Context, tenant string) error
}

// ErrNotice is a notice that isn't one line of plain text ending within the week.
var ErrNotice = errors.New("ask: a notice is one line of plain text of at most 200 characters, ending within 7 days")

// CheckNotice is nil for text a notice may say, ending at until, set at now.
func CheckNotice(text string, until, now time.Time) error {
	t := strings.TrimSpace(text)
	switch {
	case t == "", utf8.RuneCountInString(t) > MaxNoticeText, !utf8.ValidString(t),
		strings.ContainsAny(t, "<>[]`\r\n"), strings.Contains(t, "://"),
		!until.After(now), until.Sub(now) > MaxNoticeFor:
		return ErrNotice
	}
	for _, r := range t {
		if r < ' ' || r == 0x7f {
			return ErrNotice
		}
	}
	return nil
}

// Standing is the tenant's notice while it stands at now; ok false once it has ended or
// when there is none.
func Standing(ctx context.Context, s NoticeStore, tenant string, now time.Time) (CellNotice, bool, error) {
	if s == nil {
		return CellNotice{}, false, nil
	}
	n, ok, err := s.Notice(ctx, tenant)
	if err != nil || !ok || !n.Until.After(now) {
		return CellNotice{}, false, err
	}
	return n, true, nil
}

// NoticeLine is the notice as the chat apps' first line, or "" when none stands.
func NoticeLine(ctx context.Context, s NoticeStore, tenant string, now time.Time) string {
	n, ok, err := Standing(ctx, s, tenant, now)
	if err != nil || !ok {
		return ""
	}
	return "Notice: " + n.Text
}
