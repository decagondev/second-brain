package smtpmail

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"io"
	"math/big"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"sync"
	"time"
)

// Fake is a mail server for tests (SB-2307): it offers STARTTLS (unless NoTLS) with a
// certificate for 127.0.0.1 that Roots trusts, a PLAIN sign-in when User is set, refuses
// recipients in Unknown with 550, and keeps each message it took.
type Fake struct {
	Addr    string
	Roots   *x509.CertPool
	CertPEM []byte // the certificate Roots trusts, as PEM
	NoTLS   bool
	User    string
	Pass    string
	Unknown map[string]bool

	ln   net.Listener
	cert tls.Certificate
	mu   sync.Mutex
	got  []Got
}

// Got is one message the fake took.
type Got struct {
	To      string
	Subject string
	Text    string // the body, decoded
	TLS     bool
	Raw     string
}

// NewFake starts a fake mail server on 127.0.0.1.
func NewFake() (*Fake, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "fake-smtp"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, DNSNames: []string{"localhost"},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	f := &Fake{Addr: ln.Addr().String(), Roots: roots, CertPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), Unknown: map[string]bool{}, ln: ln,
		cert: tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}}
	go f.serve()
	return f, nil
}

// Close stops the fake.
func (f *Fake) Close() { _ = f.ln.Close() }

// Messages are the messages the fake took, in order.
func (f *Fake) Messages() []Got {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Got(nil), f.got...)
}

func (f *Fake) serve() {
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.session(c)
	}
}

func (f *Fake) session(c net.Conn) {
	defer func() { _ = c.Close() }()
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))
	r, w := bufio.NewReader(c), io.Writer(c)
	say := func(s string) { _, _ = io.WriteString(w, s+"\r\n") }
	say("220 fake-smtp ready")
	secure, signedIn := false, false
	var to []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		verb := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
		switch {
		case verb == "EHLO" || verb == "HELO":
			ext := []string{"fake-smtp"}
			if !secure && !f.NoTLS {
				ext = append(ext, "STARTTLS")
			}
			if secure && f.User != "" {
				ext = append(ext, "AUTH PLAIN")
			}
			for i, e := range ext {
				sep := "-"
				if i == len(ext)-1 {
					sep = " "
				}
				say("250" + sep + e)
			}
		case verb == "STARTTLS" && !secure && !f.NoTLS:
			say("220 go ahead")
			tc := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{f.cert}, MinVersion: tls.VersionTLS12})
			if tc.Handshake() != nil {
				return
			}
			c, secure = tc, true
			r, w = bufio.NewReader(tc), tc
		case verb == "AUTH":
			parts := strings.Fields(line)
			raw, _ := base64.StdEncoding.DecodeString(parts[len(parts)-1])
			if secure && string(raw) == "\x00"+f.User+"\x00"+f.Pass {
				signedIn = true
				say("235 ok")
			} else {
				say("535 no")
			}
		case verb == "MAIL":
			if f.User != "" && !signedIn {
				say("530 sign in first")
				continue
			}
			to = nil
			say("250 ok")
		case verb == "RCPT":
			addr := strings.Trim(line[strings.Index(line, ":")+1:], "<> ")
			if f.Unknown[addr] {
				say("550 no such user")
				continue
			}
			to = append(to, addr)
			say("250 ok")
		case verb == "DATA":
			say("354 go ahead")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				b.WriteString(strings.TrimPrefix(l, "."))
			}
			f.take(to, b.String(), secure)
			say("250 taken")
		case verb == "RSET" || verb == "NOOP":
			say("250 ok")
		case verb == "QUIT":
			say("221 bye")
			return
		default:
			say("502 not here")
		}
	}
}

func (f *Fake) take(to []string, raw string, secure bool) {
	g := Got{To: strings.Join(to, ","), Raw: raw, TLS: secure}
	if m, err := mail.ReadMessage(strings.NewReader(raw)); err == nil {
		g.Subject = m.Header.Get("Subject")
		body := m.Body
		if strings.EqualFold(m.Header.Get("Content-Transfer-Encoding"), "quoted-printable") {
			body = quotedprintable.NewReader(m.Body)
		}
		b, _ := io.ReadAll(body)
		g.Text = string(b)
	}
	f.mu.Lock()
	f.got = append(f.got, g)
	f.mu.Unlock()
}
