// Package syslogtls sends audit rows to a SIEM as syslog over TLS (RFC 5425; sb-cell(5)
// audit.stream; SB-2302), and keeps the stream's cursor in a file under the cell's state.
package syslogtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"example.com/second-brain/internal/core/auditstream"
)

// Timeout bounds a dial and each write.
const Timeout = 10 * time.Second

// Parse checks an audit.stream URL, syslog+tls://HOST:PORT, and returns HOST:PORT.
func Parse(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "syslog+tls" || u.Hostname() == "" || u.Port() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" {
		return "", fmt.Errorf("syslogtls: audit.stream must be syslog+tls://HOST:PORT")
	}
	if p, err := strconv.Atoi(u.Port()); err != nil || p < 1 || p > 65535 {
		return "", fmt.Errorf("syslogtls: audit.stream has a bad port")
	}
	return u.Host, nil
}

// Sender keeps one TLS connection to the SIEM, dialling again after a failure. The SIEM's
// certificate is verified for the host's name, TLS 1.2 at least, against Roots, or nil for
// the system's, which on a self-hosted cell include the company's CA in /etc/sb/ca
// (SSL_CERT_DIR); it is never trusted blindly.
type Sender struct {
	Addr  string
	Roots *x509.CertPool
	mu    sync.Mutex
	conn  net.Conn
}

// Send implements auditstream.Sender: one message, framed by octet counting.
func (s *Sender) Send(ctx context.Context, msg []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil && !alive(s.conn) {
		_ = s.conn.Close() // the SIEM hung up: a write now would vanish into a closed socket
		s.conn = nil
	}
	if s.conn == nil {
		host, _, err := net.SplitHostPort(s.Addr)
		if err != nil {
			return err
		}
		d := tls.Dialer{NetDialer: &net.Dialer{Timeout: Timeout}, Config: &tls.Config{RootCAs: s.Roots, ServerName: host, MinVersion: tls.VersionTLS12}}
		dctx, cancel := context.WithTimeout(ctx, Timeout)
		c, err := d.DialContext(dctx, "tcp", s.Addr)
		cancel()
		if err != nil {
			return err
		}
		s.conn = c
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(Timeout))
	if _, err := s.conn.Write(auditstream.Frame(msg)); err != nil {
		_ = s.conn.Close()
		s.conn = nil
		return err
	}
	return nil
}

// alive reports whether the SIEM still holds the connection open: it never writes to us,
// so a read that times out at once means alive, and anything else (its close, a reset)
// means gone. A timed-out read leaves a TLS connection usable.
func alive(c net.Conn) bool {
	_ = c.SetReadDeadline(time.Now().Add(time.Millisecond))
	var b [1]byte
	_, err := c.Read(b[:])
	_ = c.SetReadDeadline(time.Time{})
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// Close ends the connection.
func (s *Sender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil && !alive(s.conn) {
		_ = s.conn.Close() // the SIEM hung up: a write now would vanish into a closed socket
		s.conn = nil
	}
	if s.conn == nil {
		return nil
	}
	err := s.conn.Close()
	s.conn = nil
	return err
}

// FileCursor keeps the stream's cursor in a file, written whole and renamed into place.
type FileCursor struct {
	Path string
}

type cursorFile struct {
	Seq int64 `json:"seq"`
}

// Load implements auditstream.Cursor.
func (f FileCursor) Load(context.Context) (int64, error) {
	b, err := os.ReadFile(f.Path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var c cursorFile
	if err := json.Unmarshal(b, &c); err != nil || c.Seq < 0 {
		return 0, fmt.Errorf("syslogtls: %s is unreadable", f.Path) // never a silent restart from row 1
	}
	return c.Seq, nil
}

// Save implements auditstream.Cursor.
func (f FileCursor) Save(_ context.Context, seq int64) error {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(cursorFile{Seq: seq})
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.Path)
}
