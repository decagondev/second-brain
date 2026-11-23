// Package syslogtest is a fake SIEM for tests: a TLS syslog receiver (RFC 5425 framing)
// that keeps every message it takes, and can go down and come back on the same address.
package syslogtest

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

// Receiver is a fake SIEM.
type Receiver struct {
	Addr  string
	Roots *x509.CertPool // trusts the receiver's certificate, and only it
	cfg   *tls.Config
	mu    sync.Mutex
	ln    net.Listener
	conns []net.Conn
	got   []string
	wg    sync.WaitGroup
}

// New starts a receiver on a local port.
func New(t testing.TB) *Receiver {
	t.Helper()
	// httptest's certificate is for 127.0.0.1; only its TLS configuration is used.
	srv := httptest.NewUnstartedServer(nil)
	srv.StartTLS()
	r := &Receiver{cfg: srv.TLS.Clone(), Roots: x509.NewCertPool()}
	r.Roots.AddCert(srv.Certificate())
	srv.Close()
	r.cfg.NextProtos = nil
	if err := r.listen("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Down)
	return r
}

func (r *Receiver) listen(addr string) error {
	ln, err := tls.Listen("tcp", addr, r.cfg)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.ln, r.Addr = ln, ln.Addr().String()
	r.mu.Unlock()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			r.mu.Lock()
			r.conns = append(r.conns, c)
			r.mu.Unlock()
			r.wg.Add(1)
			go func() { defer r.wg.Done(); r.read(c) }()
		}
	}()
	return nil
}

// read takes octet-counted frames until the connection ends.
func (r *Receiver) read(c net.Conn) {
	br := bufio.NewReader(c)
	for {
		n, err := br.ReadString(' ')
		if err != nil {
			return
		}
		size, err := strconv.Atoi(n[:len(n)-1])
		if err != nil || size < 1 || size > 1<<20 {
			return
		}
		msg := make([]byte, size)
		if _, err := io.ReadFull(br, msg); err != nil {
			return
		}
		r.mu.Lock()
		r.got = append(r.got, string(msg))
		r.mu.Unlock()
	}
}

// Down closes the listener and every connection.
func (r *Receiver) Down() {
	r.mu.Lock()
	if r.ln != nil {
		_ = r.ln.Close()
		r.ln = nil
	}
	for _, c := range r.conns {
		_ = c.Close()
	}
	r.conns = nil
	r.mu.Unlock()
	r.wg.Wait()
}

// Up listens again on the same address.
func (r *Receiver) Up() error { return r.listen(r.Addr) }

// Got is every message taken so far, in order.
func (r *Receiver) Got() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.got...)
}
