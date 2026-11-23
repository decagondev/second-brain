package jobsvc

import (
	"context"
	"net/url"
	"path/filepath"
	"time"

	"example.com/second-brain/internal/adapters/audit/syslogtls"
	"example.com/second-brain/internal/core/auditstream"
	"example.com/second-brain/internal/core/canary"
)

// StreamEvery is how often the stream looks for new rows; after a failure it waits
// StreamRetry before trying the SIEM again.
const (
	StreamEvery = 2 * time.Second
	StreamRetry = 30 * time.Second
)

// AuditStream is the cell's stream to the SIEM (sb-audit(7), Streaming to a SIEM;
// SB-2302), or ok=false when sb-cell(5) sets no audit.stream. Its cursor is under the
// cell's state, beside the audit log it reads.
func (s *Service) AuditStream(send auditstream.Sender) (auditstream.Stream, bool) {
	c := s.Cell
	if c.Config.Audit.Stream == "" {
		return auditstream.Stream{}, false
	}
	if send == nil {
		addr, _ := syslogtls.Parse(c.Config.Audit.Stream) // checked when the config loaded
		send = &syslogtls.Sender{Addr: addr}
	}
	return auditstream.Stream{Tenant: c.Config.Tenant, Host: host(c.Config.BaseURL), Store: c.Audit.Store,
		Cursor: syslogtls.FileCursor{Path: filepath.Join(c.Config.State, "audit", "stream.json")}, Send: send, Now: s.Now}, true
}

// StreamStep sends what is new and raises the alerts: Sev-2 once the oldest row still to
// send is an hour old, again each hour it stays so, and when rows older than a day were
// skipped. It returns when the stream should look again.
func (s *Service) StreamStep(ctx context.Context, st auditstream.Stream, lastAlert *time.Time) time.Duration {
	c := s.Cell
	res, err := st.Step(ctx)
	c.Metrics.Add("sb_audit_stream_rows_total", "Audit rows the SIEM took (SB-2302).", nil, float64(res.Sent))
	c.Metrics.Set("sb_audit_stream_behind_seconds", "How old the oldest audit row still to send is.", nil, res.Behind.Seconds())
	if res.Skipped > 0 {
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev2", Kind: "audit_stream_gap", Tenant: c.Config.Tenant, Where: "audit.stream", Ref: "SB-2302"})
	}
	if res.Behind >= auditstream.Behind && s.now().Sub(*lastAlert) >= auditstream.Behind {
		*lastAlert = s.now()
		_ = c.Alerter.Alert(ctx, canary.Alert{Severity: "sev2", Kind: "audit_stream_behind", Tenant: c.Config.Tenant, Where: "audit.stream", Ref: "SB-2302"})
	}
	if err != nil {
		c.Log.Error("audit stream", "error", err.Error())
		return StreamRetry
	}
	return StreamEvery
}

// Stream runs the stream until ctx ends.
func (s *Service) Stream(ctx context.Context) {
	st, ok := s.AuditStream(nil)
	if !ok {
		return
	}
	if cl, ok := st.Send.(interface{ Close() error }); ok {
		defer func() { _ = cl.Close() }()
	}
	var lastAlert time.Time
	for {
		wait := s.StreamStep(ctx, st, &lastAlert)
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

// host is the base URL's host name, the messages' HOSTNAME.
func host(base string) string {
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
