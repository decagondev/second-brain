// Package auditstream forwards a cell's audit rows to the company's SIEM as they are
// written (sb-audit(7), Streaming to a SIEM; SB-2302). The audit log itself is the queue:
// the stream keeps only a cursor, the seq of the last row the SIEM took, so a SIEM that's
// down never blocks a write, and a restart resumes from the cursor. Each row goes out as
// one RFC 5424 message whose text is the row exactly as the export has it, so the SIEM
// can check the chain (LAW-11), and nothing but the row (LAW-07).
package auditstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"example.com/second-brain/internal/core/audit"
)

// Bounds.
const (
	Hold   = 24 * time.Hour // rows older than this when the SIEM comes back are skipped, named in a gap message
	Behind = time.Hour      // the oldest row still to send is this old: a Sev-2 alert
	Batch  = 500
)

// AppName and the structured-data id are fixed: the id uses the documentation enterprise
// number (RFC 5612), so it claims no one's.
const (
	AppName = "second-brain"
	sdID    = "sb@32473"
)

// Cursor keeps the seq of the last row the SIEM took.
type Cursor interface {
	Load(ctx context.Context) (int64, error) // 0 when nothing was sent yet
	Save(ctx context.Context, seq int64) error
}

// Sender delivers one message, or fails; a failure leaves the row to send again.
type Sender interface {
	Send(ctx context.Context, msg []byte) error
}

// Stream is one cell's stream.
type Stream struct {
	Tenant string
	Host   string // the message's HOSTNAME: the cell's host name
	Store  audit.Store
	Cursor Cursor
	Send   Sender
	Now    func() time.Time
}

// Result is what one Step did.
type Result struct {
	Sent    int
	Skipped int           // rows older than Hold, named in a gap message
	Behind  time.Duration // how old the oldest row still to send is; 0 when caught up
}

// Step sends every row after the cursor, in order, saving the cursor after each one the
// SIEM took. It stops at the first failure, leaving that row for the next Step.
func (s Stream) Step(ctx context.Context) (Result, error) {
	var res Result
	now := s.now()
	last, err := s.Cursor.Load(ctx)
	if err != nil {
		return res, fmt.Errorf("auditstream: cursor: %w", err)
	}
	for {
		rows, err := s.Store.Range(ctx, last+1, Batch)
		if err != nil {
			return res, fmt.Errorf("auditstream: audit log: %w", err)
		}
		if len(rows) == 0 {
			return res, nil
		}
		// Rows the SIEM missed for more than a day aren't sent: one gap message names them,
		// so the SIEM knows to read them from the export.
		i := 0
		for i < len(rows) && now.Sub(at(rows[i])) > Hold {
			i++
		}
		if i > 0 {
			if err := s.Send.Send(ctx, s.gap(rows[0].Seq, rows[i-1].Seq, now)); err != nil {
				res.Behind = now.Sub(at(rows[0]))
				return res, fmt.Errorf("auditstream: send: %w", err)
			}
			if err := s.Cursor.Save(ctx, rows[i-1].Seq); err != nil {
				return res, fmt.Errorf("auditstream: cursor: %w", err)
			}
			res.Skipped += i
			last = rows[i-1].Seq
		}
		for _, r := range rows[i:] {
			msg, err := s.Message(r)
			if err != nil {
				return res, err
			}
			if err := s.Send.Send(ctx, msg); err != nil {
				res.Behind = now.Sub(at(r))
				return res, fmt.Errorf("auditstream: send: %w", err)
			}
			if err := s.Cursor.Save(ctx, r.Seq); err != nil {
				return res, fmt.Errorf("auditstream: cursor: %w", err)
			}
			res.Sent++
			last = r.Seq
		}
		if len(rows) < Batch {
			return res, nil
		}
	}
}

func (s Stream) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func at(r audit.Row) time.Time {
	t, err := time.Parse(time.RFC3339Nano, r.At)
	if err != nil {
		return time.Time{} // unreadable: treated as old, so it is named in a gap, never dropped silently
	}
	return t
}

// Message is a row as an RFC 5424 message: facility log audit (13), severity info, the
// row's time, kind as MSGID, its seq and hash as structured data, and the row's JSON, as
// the export writes it, as the text.
func (s Stream) Message(r audit.Row) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		return nil, fmt.Errorf("auditstream: row %d: %w", r.Seq, err)
	}
	body := bytes.TrimRight(b.Bytes(), "\n")
	head := fmt.Sprintf("<%d>1 %s %s %s - %s [%s tenant=\"%s\" seq=\"%d\" hash=\"%s\"] ", 13*8+6, r.At, field(s.Host), AppName, field(r.Kind), sdID, sdValue(r.Tenant), r.Seq, sdValue(r.Hash))
	return append([]byte(head), body...), nil
}

// gap is a notice (severity warning) that rows from to to weren't sent.
func (s Stream) gap(from, to int64, now time.Time) []byte {
	return []byte(fmt.Sprintf("<%d>1 %s %s %s - gap [%s tenant=\"%s\" from=\"%d\" to=\"%d\"] rows %d to %d were older than a day when the SIEM came back: read them from the export",
		13*8+4, now.Format("2006-01-02T15:04:05.000Z"), field(s.Host), AppName, sdID, sdValue(s.Tenant), from, to, from, to))
}

// field is a header field: printable ASCII without spaces, at most 48 bytes, or "-".
func field(v string) string {
	var b []byte
	for i := 0; i < len(v) && len(b) < 48; i++ {
		if c := v[i]; c > 32 && c < 127 {
			b = append(b, c)
		}
	}
	if len(b) == 0 {
		return "-"
	}
	return string(b)
}

// sdValue escapes a structured-data value: ", \ and ] are escaped (RFC 5424, 6.3.3).
func sdValue(v string) string {
	var b bytes.Buffer
	for _, r := range v {
		if r == '"' || r == '\\' || r == ']' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Frame is a message framed for TLS transport by octet counting (RFC 5425): its length,
// a space, the message.
func Frame(msg []byte) []byte {
	return append([]byte(strconv.Itoa(len(msg))+" "), msg...)
}
