package mmapp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6455's accept key is SHA-1 by definition, not a security hash
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The bot's websocket (sb-ask(3), The chat app (Mattermost); SB-6005): Mattermost's
// outgoing webhooks carry no reactions and no direct messages, so the app keeps the bot's
// websocket (/api/v4/websocket) open, signs in with the bot's token in the upgrade, and
// hands each event to the app. A small RFC 6455 client, standard library only: text
// frames, continuation, ping and close. A dropped connection reconnects with backoff.

// WSEvent is one websocket event: its name, its data (Mattermost sends post and reaction
// as JSON inside strings), and the channel it was broadcast to.
type WSEvent struct {
	Event     string                     `json:"event"`
	Data      map[string]json.RawMessage `json:"data"`
	Broadcast struct {
		ChannelID string `json:"channel_id"`
	} `json:"broadcast"`
}

// Str is a data field that is a string, "" otherwise.
func (e WSEvent) Str(k string) string {
	var s string
	_ = json.Unmarshal(e.Data[k], &s)
	return s
}

// Socket is the bot's websocket.
type Socket struct {
	Base  string // the Mattermost server, https://chat.acme.example
	Token func(ctx context.Context) (string, error)
	// On is called for each event, one after another.
	On func(ctx context.Context, e WSEvent)
	// TLS is the client's TLS settings (a test server's roots); nil is the system's.
	TLS *tls.Config
	// Backoff is the first wait after a dropped connection, doubling to MaxBackoff; zero is
	// a second. Connected counts each connection made, for tests and metrics.
	Backoff   time.Duration
	Connected func()
}

// MaxBackoff is the longest wait between reconnects.
const MaxBackoff = time.Minute

// Run keeps the websocket open until ctx ends.
func (s *Socket) Run(ctx context.Context) {
	wait := s.Backoff
	if wait <= 0 {
		wait = time.Second
	}
	first := wait
	for ctx.Err() == nil {
		start := time.Now()
		err := s.once(ctx)
		if ctx.Err() != nil {
			return
		}
		if err == nil || time.Since(start) > MaxBackoff {
			wait = first // a connection that lasted starts the backoff over
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = min(wait*2, MaxBackoff)
	}
}

// ErrHandshake is an upgrade the server refused.
var ErrHandshake = errors.New("mmapp: the websocket upgrade was refused")

func (s *Socket) once(ctx context.Context) error {
	tok, err := s.Token(ctx)
	if err != nil || tok == "" {
		return fmt.Errorf("mmapp: no bot token: %w", err)
	}
	u, err := url.Parse(strings.TrimSuffix(s.Base, "/") + "/api/v4/websocket")
	if err != nil {
		return err
	}
	host := u.Host
	if u.Port() == "" {
		host += map[string]string{"https": ":443", "http": ":80"}[u.Scheme]
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	switch u.Scheme {
	case "https":
		cfg := s.TLS
		if cfg == nil {
			cfg = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		cfg = cfg.Clone()
		cfg.ServerName = u.Hostname()
		conn, err = (&tls.Dialer{NetDialer: &d, Config: cfg}).DialContext(ctx, "tcp", host)
	case "http":
		conn, err = d.DialContext(ctx, "tcp", host)
	default:
		return fmt.Errorf("mmapp: base %q isn't http or https", s.Base)
	}
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	key := make([]byte, 16)
	_, _ = rand.Read(key)
	k := base64.StdEncoding.EncodeToString(key)
	req := "GET " + u.RequestURI() + " HTTP/1.1\r\nHost: " + u.Host + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + k + "\r\nSec-WebSocket-Version: 13\r\nAuthorization: Bearer " + tok + "\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		return err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols || resp.Header.Get("Sec-WebSocket-Accept") != acceptKey(k) {
		return ErrHandshake
	}
	if s.Connected != nil {
		s.Connected()
	}
	for {
		op, msg, err := readMessage(br, conn)
		if err != nil {
			return err
		}
		switch op {
		case opClose:
			_ = writeFrame(conn, opClose, nil)
			return nil
		case opText:
			var e WSEvent
			if json.Unmarshal(msg, &e) == nil && e.Event != "" && s.On != nil {
				s.On(ctx, e)
			}
		}
	}
}

// acceptKey is RFC 6455's Sec-WebSocket-Accept for key.
func acceptKey(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")) //nolint:gosec
	return base64.StdEncoding.EncodeToString(h[:])
}

const (
	opCont  = 0x0
	opText  = 0x1
	opClose = 0x8
	opPing  = 0x9
	opPong  = 0xA
)

// MaxMessage is the largest message read; a larger one closes the connection.
const MaxMessage = 1 << 20

// readMessage reads one whole message, answering pings on the way.
func readMessage(r *bufio.Reader, w io.Writer) (byte, []byte, error) {
	var msg []byte
	var op byte
	for {
		fin, o, payload, err := readFrame(r)
		if err != nil {
			return 0, nil, err
		}
		switch o {
		case opPing:
			if err := writeFrame(w, opPong, payload); err != nil {
				return 0, nil, err
			}
			continue
		case opPong:
			continue
		case opClose:
			return opClose, nil, nil
		case opCont:
		default:
			op = o
		}
		msg = append(msg, payload...)
		if len(msg) > MaxMessage {
			return 0, nil, errors.New("mmapp: websocket message too large")
		}
		if fin {
			return op, msg, nil
		}
	}
}

func readFrame(r *bufio.Reader) (fin bool, op byte, payload []byte, err error) {
	var h [2]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return false, 0, nil, err
	}
	fin, op = h[0]&0x80 != 0, h[0]&0x0F
	masked := h[1]&0x80 != 0
	n := uint64(h[1] & 0x7F)
	switch n {
	case 126:
		var b [2]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return false, 0, nil, err
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return false, 0, nil, err
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	if n > MaxMessage {
		return false, 0, nil, errors.New("mmapp: websocket frame too large")
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(r, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload = make([]byte, n)
	if _, err := io.ReadFull(r, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return fin, op, payload, nil
}

// writeFrame writes one masked frame, as a client must.
func writeFrame(w io.Writer, op byte, payload []byte) error {
	b := []byte{0x80 | op}
	switch n := len(payload); {
	case n < 126:
		b = append(b, 0x80|byte(n))
	case n <= 0xFFFF:
		b = append(b, 0x80|126, byte(n>>8), byte(n))
	default:
		b = append(b, 0x80|127)
		b = binary.BigEndian.AppendUint64(b, uint64(n))
	}
	var mask [4]byte
	_, _ = rand.Read(mask[:])
	b = append(b, mask[:]...)
	for i, c := range payload {
		b = append(b, c^mask[i%4])
	}
	_, err := w.Write(b)
	return err
}
