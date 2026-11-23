package smtpmail

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/notify"
)

func fake(t *testing.T) *Fake {
	t.Helper()
	f, err := NewFake()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.Close)
	return f
}

func sender(f *Fake) Mail {
	return Mail{Server: f.Addr, From: "brain@acme-logistics.example", Roots: f.Roots,
		Now: func() time.Time { return time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC) }}
}

// w20 are the messages the notify job and the access request send, one of each kind.
var w20 = []notify.Message{
	{Kind: notify.KindQuestions, Text: "2 new questions for the finance space", Link: "https://brain.acme-logistics.example/?steward=finance"},
	{Kind: notify.KindReplies, Text: "1 of your questions has a reply", Link: "https://brain.acme-logistics.example/?tab=questions"},
	{Kind: notify.KindDigest, Text: "2 documents changed in your spaces this week: Refund policy; Carrier contracts", Link: "https://brain.acme-logistics.example/?tab=documents"},
	{Kind: notify.KindWatch, Text: "The sources of 1 of your saved questions changed", Link: "https://brain.acme-logistics.example/?tab=saved"},
	{Kind: notify.KindAccess, Text: "ann@acme-logistics.example asked a question that your document Carrier contracts answers. Consider sharing it with them.", Link: "https://drive.example/d/1"},
}

// contentRule is LAW-07 for one email: it holds none of the asked or answered text, and
// nothing beyond the notification's own text, its link and the fixed lines around them.
func contentRule(g Got, m notify.Message, secret []string) error {
	for _, s := range secret {
		if strings.Contains(g.Raw, s) || strings.Contains(g.Text, s) {
			return fmt.Errorf("the email to %s holds %q", g.To, s)
		}
	}
	if g.Text != Text(m) {
		return fmt.Errorf("the email to %s holds more than its notification: %q", g.To, g.Text)
	}
	return nil
}

func TestEachKindGoesOverTLS(t *testing.T) {
	f := fake(t)
	send := sender(f)
	for i, m := range w20 {
		if err := send.Send(context.Background(), notify.Person{ID: "usr_ann", Email: "ann@acme-logistics.example"}, m); err != nil {
			t.Fatalf("%s: %v", m.Kind, err)
		}
		got := f.Messages()
		if len(got) != i+1 {
			t.Fatalf("%s: the server took %d messages", m.Kind, len(got))
		}
		g := got[i]
		if !g.TLS || g.To != "ann@acme-logistics.example" || g.Subject != "Second Brain: "+Subjects[m.Kind] {
			t.Fatalf("%s: tls=%v to=%q subject=%q", m.Kind, g.TLS, g.To, g.Subject)
		}
		if !strings.Contains(g.Text, m.Text) || !strings.Contains(g.Text, m.Link) {
			t.Fatalf("%s: the text lost its content: %q", m.Kind, g.Text)
		}
		if err := contentRule(g, m, []string{"CANARY"}); err != nil {
			t.Fatal(err)
		}
	}
}

// The check catches a message that carries a document's text, as a "helpful" preview would.
func TestContentRuleCatchesDocumentText(t *testing.T) {
	f := fake(t)
	doc := "CANARY refunds are paid within 30 days of the claim"
	m := notify.Message{Kind: notify.KindWatch, Text: "The sources of 1 of your saved questions changed: " + doc, Link: "https://brain.acme-logistics.example/?tab=saved"}
	if err := sender(f).Send(context.Background(), notify.Person{Email: "ann@acme-logistics.example"}, m); err != nil {
		t.Fatal(err)
	}
	got := f.Messages()
	if len(got) != 1 {
		t.Fatalf("got %d", len(got))
	}
	if err := contentRule(got[0], w20[3], []string{"CANARY"}); err == nil {
		t.Fatal("an email with document text passed the content rule")
	}
}

func TestRefusesAServerWithoutSTARTTLS(t *testing.T) {
	f := fake(t)
	f.NoTLS = true
	err := sender(f).Send(context.Background(), notify.Person{Email: "ann@acme-logistics.example"}, w20[0])
	if !errors.Is(err, ErrNoTLS) {
		t.Fatalf("err = %v, want ErrNoTLS", err)
	}
	if n := len(f.Messages()); n != 0 {
		t.Fatalf("%d messages went in the clear", n)
	}
}

func TestVerifiesTheServersCertificate(t *testing.T) {
	f := fake(t)
	m := sender(f)
	m.Roots = nil // the system's roots don't know the fake's certificate
	if err := m.Send(context.Background(), notify.Person{Email: "ann@acme-logistics.example"}, w20[0]); err == nil {
		t.Fatal("a server with an untrusted certificate took the message")
	}
	if n := len(f.Messages()); n != 0 {
		t.Fatalf("%d messages went to an unverified server", n)
	}
}

func TestSignIn(t *testing.T) {
	f := fake(t)
	f.User, f.Pass = "brain", "s3cret"
	m := sender(f)
	ctx := context.Background()
	to := notify.Person{Email: "ann@acme-logistics.example"}
	if err := m.Send(ctx, to, w20[0]); err == nil {
		t.Fatal("sent without signing in to a server that wants it")
	}
	m.Login = func(context.Context) (string, string, error) { return "brain", "wrong", nil }
	if err := m.Send(ctx, to, w20[0]); err == nil {
		t.Fatal("sent with a wrong password")
	}
	m.Login = func(context.Context) (string, string, error) { return "brain", "s3cret", nil }
	if err := m.Send(ctx, to, w20[0]); err != nil {
		t.Fatal(err)
	}
	if got := f.Messages(); len(got) != 1 || strings.Contains(got[0].Raw, "s3cret") {
		t.Fatalf("got %d messages, or one holding the password", len(got))
	}
}

func TestUnknownAndBadAddressesAreSkipped(t *testing.T) {
	f := fake(t)
	f.Unknown["gone@acme-logistics.example"] = true
	ctx := context.Background()
	for _, email := range []string{"gone@acme-logistics.example", "ann@acme-logistics.example\r\nBcc: eve@beta-freight.example", "Ann <ann@acme-logistics.example>", ""} {
		if err := sender(f).Send(ctx, notify.Person{Email: email}, w20[0]); !errors.Is(err, notify.ErrNoSuchPerson) {
			t.Fatalf("%q: err = %v, want ErrNoSuchPerson", email, err)
		}
	}
	if n := len(f.Messages()); n != 0 {
		t.Fatalf("%d messages went", n)
	}
}

func TestHeadersHoldNothingFromTheMessage(t *testing.T) {
	m := sender(&Fake{})
	b, err := m.Body("brain@acme-logistics.example", "ann@acme-logistics.example", notify.Message{Kind: notify.KindQuestions, Text: "2 new questions\r\nBcc: eve@beta-freight.example", Link: "https://x.example/\r\nX: y"})
	if err != nil {
		t.Fatal(err)
	}
	head := string(b[:strings.Index(string(b), "\r\n\r\n")])
	if strings.Contains(head, "Bcc") || strings.Contains(head, "X: y") || strings.Contains(string(b), "\r\nBcc:") {
		t.Fatalf("the message's text reached the headers: %q", b)
	}
	if _, err := m.Body("brain@acme-logistics.example", "ann@acme-logistics.example", notify.Message{Kind: "other"}); err == nil {
		t.Fatal("a kind with no subject was sent")
	}
}

func TestParse(t *testing.T) {
	for _, ok := range []string{"mail.acme-logistics.example:587", "127.0.0.1:25"} {
		if err := Parse(ok); err != nil {
			t.Fatalf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "mail.acme-logistics.example", "smtp://mail:587", "mail:0", "mail:x", "u@mail:587"} {
		if err := Parse(bad); err == nil {
			t.Fatalf("%q passed", bad)
		}
	}
}
