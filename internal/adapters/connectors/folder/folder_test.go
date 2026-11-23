package folder

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/second-brain/internal/core/connectorkit"
	"example.com/second-brain/internal/core/ports"
)

// faulty is the share with one fault on the next ACL file read: "500" an unreadable ACL
// file, "timeout" a share slower than the deadline, "404" the ACL file gone.
type faulty struct {
	fs.FS
	mu    sync.Mutex
	fault string
}

func (f *faulty) Open(name string) (fs.File, error) {
	if strings.HasSuffix(name, ".acl") {
		f.mu.Lock()
		fault := f.fault
		f.fault = ""
		f.mu.Unlock()
		switch fault {
		case "500":
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
		case "timeout":
			time.Sleep(ACLDeadline + 200*time.Millisecond)
		case "404":
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	return f.FS.Open(name)
}

type sandbox struct {
	dir string
	f   *faulty
	c   *Connector
	n   int
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func newSandbox(t *testing.T) *sandbox {
	dir := t.TempDir()
	write(t, dir, "finance/q3-forecast.md", "# Q3 forecast\n\nRevenue grows 4% on the freight lanes.")
	write(t, dir, "finance/q3-forecast.md.acl", "read ann@acme.example\nlabel confidential\n")
	for i := 0; i < 200; i++ {
		write(t, dir, fmt.Sprintf("many/doc-%03d.txt", i), fmt.Sprintf("document %d", i))
	}
	write(t, dir, "many/.acl", "read domain:acme.example\n")
	c, err := Open(dir, filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	f := &faulty{FS: c.FS}
	c.FS = f
	return &sandbox{dir: dir, f: f, c: c}
}

func (s *sandbox) Connector() ports.Connector { return s.c }
func (s *sandbox) Doc() ports.SourceRef {
	return ports.SourceRef{Kind: "docs", ID: "finance/q3-forecast.md"}
}
func (s *sandbox) Allowed() ports.UserRef {
	return ports.UserRef{ID: "usr_a", Email: "ann@acme.example"}
}
func (s *sandbox) Denied() ports.UserRef {
	return ports.UserRef{ID: "usr_b", Email: "bob@acme.example"}
}
func (s *sandbox) Refs(n int) []ports.SourceRef {
	out := make([]ports.SourceRef, n)
	for i := range out {
		out[i] = ports.SourceRef{Kind: "docs", ID: fmt.Sprintf("many/doc-%03d.txt", i%200)}
	}
	return out
}
func (s *sandbox) InjectACLFault(kind string) { s.f.mu.Lock(); s.f.fault = kind; s.f.mu.Unlock() }
func (s *sandbox) bump(name string) {
	s.n++
	p := filepath.Join(s.dir, filepath.FromSlash(name))
	at := time.Now().Add(time.Duration(s.n) * time.Hour)
	_ = os.Chtimes(p, at, at)
}
func (s *sandbox) Edit(ref ports.SourceRef) {
	p := filepath.Join(s.dir, filepath.FromSlash(ref.ID))
	b, _ := os.ReadFile(p)
	_ = os.WriteFile(p, append(b, " Edited."...), 0o600)
	s.bump(ref.ID)
}
func (s *sandbox) ChangePermissions(ref ports.SourceRef) {
	p := filepath.Join(s.dir, filepath.FromSlash(ref.ID)+".acl")
	b, _ := os.ReadFile(p)
	_ = os.WriteFile(p, append(b, fmt.Sprintf("read reader%d@acme.example\n", s.n)...), 0o600)
	s.n++
}
func (s *sandbox) Action() (ports.Action, bool) { return ports.Action{}, false }
func (s *sandbox) Effects() int                 { return 0 }

func TestContractKit(t *testing.T) { connectorkit.Run(t, newSandbox(t)) }

// SB-1606: a file with no ACL file is Unknown, never allowed (LAW-03), and so is an ACL
// file that can't be parsed.
func TestNoACLIsUnknown(t *testing.T) {
	s := newSandbox(t)
	write(t, s.dir, "loose/notes.md", "no one said who may read this")
	write(t, s.dir, "odd/plan.md", "a plan")
	write(t, s.dir, "odd/plan.md.acl", "read ann@acme.example\nallow everyone\n")
	vs, err := s.c.ACL(context.Background(), s.Allowed(), []ports.SourceRef{{Kind: "docs", ID: "loose/notes.md"}, {Kind: "docs", ID: "odd/plan.md"},
		{Kind: "docs", ID: "gone.md"}, {Kind: "docs", ID: "../etc/passwd"}, {Kind: "docs", ID: "many/doc-001.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []ports.Verdict{ports.VerdictUnknown, ports.VerdictUnknown, ports.VerdictUnknown, ports.VerdictUnknown, ports.VerdictAllow}
	for i := range want {
		if vs[i] != want[i] {
			t.Errorf("ref %d: %v, want %v", i, vs[i], want[i])
		}
	}
	if v, _ := s.c.ACL(context.Background(), ports.UserRef{Email: "eve@other.example"}, s.Refs(1)); v[0] != ports.VerdictDeny {
		t.Errorf("another domain: %v, want deny", v[0])
	}
}

func TestFetchAndRouting(t *testing.T) {
	s := newSandbox(t)
	d, err := s.c.Fetch(context.Background(), s.Doc())
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "q3-forecast" || d.Location != "/finance/q3-forecast" || d.Label != "confidential" ||
		!strings.Contains(d.Text, "freight lanes") || len(d.ACLPrefilter) != 1 || d.ACLPrefilter[0] != "ann@acme.example" {
		t.Fatalf("document %+v", d)
	}
	for _, bad := range []string{"../outside.md", "/etc/passwd", "finance/q3-forecast.md.acl", ".hidden/x.md", "x.exe"} {
		if _, err := s.c.Fetch(context.Background(), ports.SourceRef{Kind: "docs", ID: bad}); err == nil {
			t.Errorf("%q was fetched", bad)
		}
	}
}

// A link inside the share that points outside it is never followed.
func TestLinksStayInside(t *testing.T) {
	outside := t.TempDir()
	write(t, outside, "secret.md", "outside the share")
	s := newSandbox(t)
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(s.dir, "finance", "link.md")); err != nil {
		t.Skip("no symlinks here:", err)
	}
	if _, err := s.c.Fetch(context.Background(), ports.SourceRef{Kind: "docs", ID: "finance/link.md"}); err == nil {
		t.Fatal("a link out of the share was read")
	}
}

func TestDeletesAndLostCursors(t *testing.T) {
	s := newSandbox(t)
	ctx := context.Background()
	all, cur, err := s.c.Changes(ctx, "")
	if err != nil || len(all) != 201 {
		t.Fatalf("first listing: %d changes, %v", len(all), err)
	}
	if err := os.Remove(filepath.Join(s.dir, "many", "doc-007.txt")); err != nil {
		t.Fatal(err)
	}
	set, _, err := s.c.Changes(ctx, cur)
	if err != nil || len(set) != 1 || !set[0].Deleted || set[0].Ref.ID != "many/doc-007.txt" {
		t.Fatalf("after a delete: %+v %v", set, err)
	}
	if _, _, err := s.c.Changes(ctx, ports.Cursor("l1:"+strings.Repeat("0", 24))); err == nil {
		t.Fatal("a cursor whose listing is gone started afresh")
	}
	if _, _, err := s.c.Changes(ctx, "page-token"); err == nil {
		t.Fatal("a foreign cursor was accepted")
	}
	for i := 0; i < Keep+3; i++ { // older listings are pruned
		write(t, s.dir, fmt.Sprintf("new/%d.md", i), "x")
		if _, _, err := s.c.Changes(ctx, ""); err != nil {
			t.Fatal(err)
		}
	}
	if ms, _ := filepath.Glob(filepath.Join(s.c.State, "*.json")); len(ms) != Keep {
		t.Fatalf("%d listings kept, want %d", len(ms), Keep)
	}
}

func TestParseACL(t *testing.T) {
	a, err := parseACL([]byte("# finance\nread Ann@Acme.example\nread domain:beta-freight.example # partners\nowner cfo@acme.example\nlabel restricted\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !a.allows("ann@acme.example") || !a.allows("x@beta-freight.example") || !a.allows("CFO@acme.example") || a.allows("bob@acme.example") || a.label != "restricted" {
		t.Fatalf("acl %+v", a)
	}
	for _, bad := range []string{"read", "read everyone", "read domain:", "owner a@b.c\nowner d@e.f", "label Top Secret", "write ann@acme.example", "read a@b@c"} {
		if _, err := parseACL([]byte(bad)); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}

func pdf(stream []byte, flate bool) []byte {
	dict := fmt.Sprintf("<< /Length %d >>", len(stream))
	if flate {
		var z bytes.Buffer
		w := zlib.NewWriter(&z)
		_, _ = w.Write(stream)
		_ = w.Close()
		stream = z.Bytes()
		dict = fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>", len(stream))
	}
	return []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n4 0 obj\n" + dict + "\nstream\n" + string(stream) + "\nendstream\nendobj\n%%EOF\n")
}

func TestPDFText(t *testing.T) {
	page := []byte("BT /F1 12 Tf 72 720 Td (Freight lanes \\(EU\\)) Tj 0 -14 Td [(Rev) -20 (enue) -300 (grows)] TJ T* (4\\045 in Q3) ' ET\n" +
		"BT <FEFF00480069> Tj ET <0102> Tj")
	for _, flate := range []bool{false, true} {
		got := PDFText(pdf(page, flate))
		want := "Freight lanes (EU)\nRevenue grows\n4% in Q3\nHi"
		if got != want {
			t.Errorf("flate=%v:\n got %q\nwant %q", flate, got, want)
		}
	}
	if PDFText([]byte("not a pdf")) != "" || PDFText(pdf([]byte("garbage"), false)) != "" {
		t.Error("text from nothing")
	}
	broken := pdf(page, true)
	broken[len(broken)-30] ^= 0xff
	_ = PDFText(broken) // never panics
}

func TestBigAndBinaryFilesAreNamesOnly(t *testing.T) {
	s := newSandbox(t)
	write(t, s.dir, "many/blob.txt", "\xff\xfe\x00binary")
	write(t, s.dir, "many/big.txt", strings.Repeat("x", MaxSize+1))
	for _, id := range []string{"many/blob.txt", "many/big.txt"} {
		d, err := s.c.Fetch(context.Background(), ports.SourceRef{Kind: "docs", ID: id})
		if err != nil || d.Text != "" || d.Title == "" {
			t.Errorf("%s: %q %v", id, d.Text, err)
		}
	}
}

// The example share the self-hosted demo mounts (examples/share): every document but its
// README has an ACL file that parses, and the PDF's text is read.
func TestExampleShare(t *testing.T) {
	c, err := Open(filepath.Join("..", "..", "..", "..", "examples", "share"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()
	set, _, err := c.Changes(ctx, "")
	if err != nil || len(set) < 5 {
		t.Fatalf("%d documents, %v", len(set), err)
	}
	ann := ports.UserRef{Email: "ann@acme.example"}
	for _, ch := range set {
		v, _ := c.ACL(ctx, ann, []ports.SourceRef{ch.Ref})
		want := ports.VerdictAllow
		switch {
		case ch.Ref.ID == "README.md":
			want = ports.VerdictUnknown
		case strings.HasPrefix(ch.Ref.ID, "people/"):
			want = ports.VerdictDeny
		}
		if v[0] != want {
			t.Errorf("%s for ann: %v, want %v", ch.Ref.ID, v[0], want)
		}
	}
	d, err := c.Fetch(ctx, ports.SourceRef{Kind: "docs", ID: "operations/driver-safety.pdf"})
	if err != nil || !strings.Contains(d.Text, "tyre pressure") {
		t.Fatalf("the PDF's text: %q %v", d.Text, err)
	}
	for id, want := range map[string]string{"operations/fuel-cards.docx": "blocked within the hour", "operations/night-shift.html": "seal number goes on the manifest"} {
		if d, err := c.Fetch(ctx, ports.SourceRef{Kind: "docs", ID: id}); err != nil || !strings.Contains(d.Text, want) {
			t.Errorf("%s: %q %v", id, d.Text, err)
		}
	}
}
