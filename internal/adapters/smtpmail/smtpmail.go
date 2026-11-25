// Package smtpmail sends notifications by email (sb-ask(3), Notifications; sb-cell(5)
// email; SB-2307), for a site with neither Slack nor Mattermost. It always says STARTTLS
// and refuses a server that doesn't offer it, verifying the server's certificate; the
// message is plain text with the same content as the chat's: a kind, counts and a link,
// never a question, an answer or a document's text (LAW-07).
package smtpmail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"example.com/second-brain/internal/core/notify"
)

// Timeout bounds one message: the dial and the whole conversation.
const Timeout = 30 * time.Second

// ErrNoTLS is a server that doesn't offer STARTTLS: refused, never used in the clear.
var ErrNoTLS = errors.New("smtpmail: the mail server doesn't offer STARTTLS")

// Parse checks an email.server, HOST:PORT.
func Parse(server string) error {
	host, port, err := net.SplitHostPort(server)
	if err != nil || host == "" || strings.ContainsAny(host, "/@ ") {
		return fmt.Errorf("smtpmail: email.server must be HOST:PORT")
	}
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("smtpmail: email.server has a bad port")
	}
	return nil
}

// Address checks a sender or recipient: one bare address, no name, no line breaks.
func Address(v string) (string, error) {
	a, err := mail.ParseAddress(v)
	if err != nil || a.Name != "" || a.Address != v || strings.ContainsAny(v, "\r\n<>") {
		return "", fmt.Errorf("smtpmail: %q is not one email address", v)
	}
	return a.Address, nil
}

// Mail sends each notification as one email.
type Mail struct {
	Server string // HOST:PORT
	From   string
	// Login reads the sign-in, if the server wants one; "" is none. Nil is none.
	Login func(ctx context.Context) (user, password string, err error)
	Roots *x509.CertPool // nil: the system's, which include /etc/sb/ca on a self-hosted cell
	Now   func() time.Time
}

var _ notify.Sender = Mail{}

// Subjects name the kind only.
var Subjects = map[string]string{
	notify.KindQuestions: "new questions",
	notify.KindReplies:   "replies to your questions",
	notify.KindDigest:    "what changed this week",
	notify.KindWatch:     "your saved questions",
	notify.KindAccess:    "a request for access",
	notify.KindTest:      "a test",
	notify.KindTokens:    "an API token expires soon",
}

// Send implements notify.Sender.
func (m Mail) Send(ctx context.Context, to notify.Person, msg notify.Message) error {
	rcpt, err := Address(to.Email)
	if err != nil {
		return notify.ErrNoSuchPerson
	}
	from, err := Address(m.From)
	if err != nil {
		return err
	}
	body, err := m.Body(from, rcpt, msg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	c, host, err := open(ctx, m.Server, m.Roots)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if m.Login != nil {
		user, pass, err := m.Login(ctx)
		if err != nil {
			return err
		}
		if user != "" {
			if err := c.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
				return fmt.Errorf("smtpmail: sign-in: %w", err)
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(rcpt); err != nil {
		var te *textproto.Error
		if errors.As(err, &te) && (te.Code == 550 || te.Code == 551 || te.Code == 553) {
			return notify.ErrNoSuchPerson
		}
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// open connects to server and says STARTTLS, verifying the server's certificate.
func open(ctx context.Context, server string, roots *x509.CertPool) (*smtp.Client, string, error) {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		return nil, "", err
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server)
	if err != nil {
		return nil, "", err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, "", err
	}
	if err := c.Hello("second-brain"); err != nil {
		_ = c.Close()
		return nil, "", err
	}
	if ok, _ := c.Extension("STARTTLS"); !ok {
		_ = c.Close()
		return nil, "", ErrNoTLS
	}
	if err := c.StartTLS(&tls.Config{ServerName: host, RootCAs: roots, MinVersion: tls.VersionTLS12}); err != nil {
		_ = c.Close()
		return nil, "", fmt.Errorf("smtpmail: STARTTLS: %w", err)
	}
	return c, host, nil
}

// Probe checks that server answers, offers STARTTLS and shows a certificate roots trust
// for its name (SB-2601); it sends nothing and signs in to nothing.
func Probe(ctx context.Context, server string, roots *x509.CertPool) error {
	if err := Parse(server); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	c, _, err := open(ctx, server, roots)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	return c.Quit()
}

// Body is the whole message: headers, then the text, the link and where to turn the kind
// off, quoted-printable. Nothing in it comes from outside the notification.
func (m Mail) Body(from, to string, msg notify.Message) ([]byte, error) {
	subject, ok := Subjects[msg.Kind]
	if !ok {
		return nil, fmt.Errorf("smtpmail: no subject for kind %q", msg.Kind)
	}
	now := time.Now
	if m.Now != nil {
		now = m.Now
	}
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	domain := from[strings.LastIndexByte(from, '@')+1:]
	var b bytes.Buffer
	for _, h := range [][2]string{
		{"From", from}, {"To", to}, {"Subject", "Second Brain: " + subject},
		{"Date", now().UTC().Format(time.RFC1123Z)}, {"Message-ID", "<" + hex.EncodeToString(id) + "@" + domain + ">"},
		{"Auto-Submitted", "auto-generated"}, {"MIME-Version", "1.0"},
		{"Content-Type", "text/plain; charset=utf-8"}, {"Content-Transfer-Encoding", "quoted-printable"},
	} {
		b.WriteString(h[0] + ": " + h[1] + "\r\n")
	}
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(Text(msg))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Text is the plain-text body: the notification's text, its link when it is https, and
// where to turn the kind off.
func Text(msg notify.Message) string {
	text := strings.NewReplacer("\r", " ", "\n", " ").Replace(msg.Text) + "\r\n"
	if strings.HasPrefix(msg.Link, "https://") && !strings.ContainsAny(msg.Link, " \r\n") {
		text += "\r\n" + msg.Link + "\r\n"
	}
	if strings.HasPrefix(msg.Pause, "https://") && !strings.ContainsAny(msg.Pause, " \r\n") { // SB-5107
		text += "\r\nPause this agent here: " + msg.Pause + "\r\n"
	}
	return text + "\r\nYou can turn off " + msg.Kind + " notifications in the web UI's notification settings.\r\n"
}
