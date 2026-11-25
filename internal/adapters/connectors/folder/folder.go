// Package folder is the `docs: folder` connector (sb-connector(3); SB-1606): the text,
// Markdown and PDF files of a directory, such as a file share mounted read-only on a
// self-hosted cell. Who may read each file is written beside it in an ACL file; a file
// with no ACL file is Unknown, never allowed (LAW-03).
package folder

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"example.com/second-brain/internal/core/ports"
)

const kind = "docs"

// ACLDeadline is the deadline for one ACL batch (sb-connector(3), ACL).
const ACLDeadline = 800 * time.Millisecond

// MaxSize is the largest file whose text is read; a larger one is indexed by name only.
const MaxSize = 8 << 20

// Keep is how many listings are kept for cursors to name.
const Keep = 8

// Connector reads FS. Open makes one rooted at a directory.
type Connector struct {
	FS    fs.FS  // the directory, as a root the connector can't leave
	State string // where listings are kept; "" keeps them in memory (tests)

	mu    sync.Mutex
	mem   map[string][]byte
	root  *os.Root
	notes *os.Root // notes/, writable, with the manifest's notes: true (SB-1705)
}

var _ ports.Connector = (*Connector)(nil)

// Open opens dir as the connector's root: a link inside it can't reach outside it.
func Open(dir, state string) (*Connector, error) {
	r, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("folder: %w", err)
	}
	return &Connector{FS: r.FS(), State: state, root: r}, nil
}

// Close releases the root.
func (c *Connector) Close() error {
	if c.notes != nil {
		_ = c.notes.Close()
	}
	if c.root != nil {
		return c.root.Close()
	}
	return nil
}

// APIVersion implements ports.Connector.
func (c *Connector) APIVersion() int { return ports.ConnectorAPIVersion }

// Document reports whether a path names a document: a .txt, .md, .pdf, .docx, .pptx,
// .odt, .html, .htm, .xlsx, .ods, .csv, .vtt or .srt file, with no part of its path starting with a dot.
func Document(p string) bool {
	if !fs.ValidPath(p) || p == "." {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".txt", ".md", ".pdf", ".docx", ".pptx", ".odt", ".html", ".htm", ".xlsx", ".ods", ".csv", ".vtt", ".srt":
		return true
	}
	return false
}

// acl is one file's ACL file, parsed.
type acl struct {
	readers []string // emails and domain:DOMAIN, lower case
	owner   string
	label   string
	sum     string // the ACL file's content hash
}

var errNoACL = errors.New("folder: the file has no ACL file")

// aclFor reads the file's own ACL file, or its folder's.
func (c *Connector) aclFor(id string) (acl, error) {
	b, err := fs.ReadFile(c.FS, id+".acl")
	if errors.Is(err, fs.ErrNotExist) {
		b, err = fs.ReadFile(c.FS, path.Join(path.Dir(id), ".acl"))
		if errors.Is(err, fs.ErrNotExist) {
			return acl{}, errNoACL
		}
	}
	if err != nil {
		return acl{}, err
	}
	return parseACL(b)
}

func parseACL(b []byte) (acl, error) {
	sum := sha256.Sum256(b)
	a := acl{sum: hex.EncodeToString(sum[:8])}
	sc := bufio.NewScanner(bytes.NewReader(b))
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		if len(f) != 2 {
			return acl{}, fmt.Errorf("folder: ACL line %d: want RULE VALUE", n)
		}
		v := strings.ToLower(f[1])
		switch f[0] {
		case "read":
			if d, ok := strings.CutPrefix(v, "domain:"); ok {
				if d == "" || strings.Contains(d, "@") {
					return acl{}, fmt.Errorf("folder: ACL line %d: bad domain", n)
				}
			} else if !email(v) {
				return acl{}, fmt.Errorf("folder: ACL line %d: bad email", n)
			}
			a.readers = append(a.readers, v)
		case "owner":
			if !email(v) || a.owner != "" {
				return acl{}, fmt.Errorf("folder: ACL line %d: one owner, by email", n)
			}
			a.owner = v
		case "label":
			for _, r := range v {
				if (r < 'a' || r > 'z') && r != '-' {
					return acl{}, fmt.Errorf("folder: ACL line %d: a label is [a-z-]", n)
				}
			}
			a.label = v
		default:
			return acl{}, fmt.Errorf("folder: ACL line %d: unknown rule %q", n, f[0])
		}
	}
	if err := sc.Err(); err != nil {
		return acl{}, err
	}
	if a.owner != "" {
		a.readers = append(a.readers, a.owner)
	}
	sort.Strings(a.readers)
	return a, nil
}

func email(v string) bool {
	at := strings.IndexByte(v, '@')
	return at > 0 && at == strings.LastIndexByte(v, '@') && at < len(v)-1 && !strings.ContainsAny(v, " ,;")
}

func (a acl) allows(e string) bool {
	e = strings.ToLower(strings.TrimSpace(e))
	at := strings.LastIndexByte(e, '@')
	if at < 0 {
		return false
	}
	for _, r := range a.readers {
		if r == e || r == "domain:"+e[at+1:] {
			return true
		}
	}
	return false
}

// Fetch implements ports.Connector.
func (c *Connector) Fetch(_ context.Context, ref ports.SourceRef) (ports.Document, error) {
	if ref.Kind != kind || !Document(ref.ID) {
		return ports.Document{}, fmt.Errorf("folder: %q isn't a document", ref.ID)
	}
	st, err := fs.Stat(c.FS, ref.ID)
	if err != nil {
		return ports.Document{}, fmt.Errorf("folder: %w", err)
	}
	if !st.Mode().IsRegular() {
		return ports.Document{}, fmt.Errorf("folder: %q isn't a file", ref.ID)
	}
	name := path.Base(ref.ID)
	title := strings.TrimSuffix(name, path.Ext(name))
	loc := "/" + strings.TrimSuffix(ref.ID, path.Ext(ref.ID))
	doc := ports.Document{Ref: ref, Version: int(st.ModTime().UnixNano()), Title: title, Location: loc}
	if a, aerr := c.aclFor(ref.ID); aerr == nil {
		doc.ACLPrefilter, doc.Label, doc.Owner = a.readers, a.label, a.owner
	}
	if st.Size() > MaxSize {
		return doc, nil
	}
	b, err := fs.ReadFile(c.FS, ref.ID)
	if err != nil {
		return ports.Document{}, fmt.Errorf("folder: %w", err)
	}
	doc.Text, _ = Text(ref.ID, b)
	doc.Pages = Pages(ref.ID, b) // SB-5205
	if e := strings.ToLower(path.Ext(ref.ID)); e == ".docx" || e == ".pptx" {
		doc.ReviewBy = OfficeReviewBy(b)
	}
	return doc, nil
}

// Text is a document's text as Fetch reads it, by its name's extension; ok is false for a
// name that isn't a document (an ask's file, SB-3802, uses it too).
func Text(name string, b []byte) (string, bool) {
	if !Document(path.Base(name)) {
		return "", false
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".pdf":
		return PDFText(b), true
	case ".vtt", ".srt": // SB-3701
		t, _ := TranscriptText(b)
		return t, true
	case ".docx", ".pptx", ".odt", ".html", ".htm", ".xlsx", ".ods", ".csv":
		return OfficeText(name, b), true
	}
	if utf8.Valid(b) {
		return string(b), true
	}
	return "", true
}

// ACL implements ports.Connector: read live from the ACL file. No ACL file, one that
// can't be read or parsed, a file that's gone, or a late answer is Unknown (LAW-03).
func (c *Connector) ACL(ctx context.Context, user ports.UserRef, refs []ports.SourceRef) ([]ports.Verdict, error) {
	out := make([]ports.Verdict, len(refs))
	if user.Email == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, ACLDeadline)
	defer cancel()
	done := make(chan []ports.Verdict, 1)
	go func() {
		vs := make([]ports.Verdict, len(refs))
		for i, r := range refs {
			if ctx.Err() != nil {
				break
			}
			vs[i] = c.verdict(user.Email, r)
		}
		done <- vs
	}()
	select {
	case vs := <-done:
		if ctx.Err() != nil {
			return out, nil
		}
		return vs, nil
	case <-ctx.Done():
		return out, nil // late is Unknown
	}
}

func (c *Connector) verdict(e string, r ports.SourceRef) ports.Verdict {
	if r.Kind != kind || !Document(r.ID) {
		return ports.VerdictUnknown
	}
	if st, err := fs.Stat(c.FS, r.ID); err != nil || !st.Mode().IsRegular() {
		return ports.VerdictUnknown
	}
	a, err := c.aclFor(r.ID)
	if err != nil {
		return ports.VerdictUnknown
	}
	if a.allows(e) {
		return ports.VerdictAllow
	}
	return ports.VerdictDeny
}

// entry is one document in a listing: its content's mark and its ACL file's.
type entry struct {
	M string `json:"m"` // modification time and size
	A string `json:"a"` // the ACL file's hash, "" for none or unreadable
}

func (c *Connector) list() (map[string]entry, error) {
	out := map[string]entry{}
	err := fs.WalkDir(c.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !Document(p) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		e := entry{M: fmt.Sprintf("%d/%d", info.ModTime().UnixNano(), info.Size())}
		if a, aerr := c.aclFor(p); aerr == nil {
			e.A = a.sum
		} else if !errors.Is(aerr, errNoACL) {
			e.A = "unreadable"
		}
		out[p] = e
		return nil
	})
	return out, err
}

// Changes implements ports.Connector: the listing now against the one the cursor names.
func (c *Connector) Changes(_ context.Context, cur ports.Cursor) (ports.ChangeSet, ports.Cursor, error) {
	now, err := c.list()
	if err != nil {
		return nil, cur, fmt.Errorf("folder: list: %w", err)
	}
	var before map[string]entry
	if cur != "" {
		before, err = c.load(string(cur))
		if err != nil {
			return nil, cur, err
		}
	}
	var set ports.ChangeSet
	for _, p := range sorted(now) {
		was, ok := before[p]
		switch {
		case !ok:
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: p}})
		case was.M != now[p].M:
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: p}})
		case was.A != now[p].A:
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: p}, ACLOnly: true})
		}
	}
	for _, p := range sorted(before) {
		if _, ok := now[p]; !ok {
			set = append(set, ports.Change{Ref: ports.SourceRef{Kind: kind, ID: p}, Deleted: true})
		}
	}
	next, err := c.save(now)
	if err != nil {
		return nil, cur, err
	}
	return set, ports.Cursor(next), nil
}

func sorted(m map[string]entry) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// save keeps a listing under its content's name and returns the cursor that names it.
func (c *Connector) save(l map[string]entry) (string, error) {
	b, _ := json.Marshal(l) // map keys marshal sorted, so equal listings are equal bytes
	s := sha256.Sum256(b)
	cur := "l1:" + hex.EncodeToString(s[:12])
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State == "" {
		if c.mem == nil {
			c.mem = map[string][]byte{}
		}
		c.mem[cur] = b
		return cur, nil
	}
	if err := os.MkdirAll(c.State, 0o700); err != nil {
		return "", err
	}
	name := filepath.Join(c.State, strings.TrimPrefix(cur, "l1:")+".json")
	if _, err := os.Stat(name); err != nil {
		tmp := name + ".tmp"
		if err := os.WriteFile(tmp, b, 0o600); err != nil {
			return "", err
		}
		if err := os.Rename(tmp, name); err != nil {
			return "", err
		}
	} else {
		now := time.Now()
		_ = os.Chtimes(name, now, now) // the newest stays newest
	}
	c.prune()
	return cur, nil
}

func (c *Connector) prune() {
	ms, _ := filepath.Glob(filepath.Join(c.State, "*.json"))
	if len(ms) <= Keep {
		return
	}
	type f struct {
		p string
		t time.Time
	}
	var files []f
	for _, m := range ms {
		if st, err := os.Stat(m); err == nil {
			files = append(files, f{m, st.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].t.After(files[j].t) })
	for _, x := range files[min(Keep, len(files)):] {
		_ = os.Remove(x.p)
	}
}

func (c *Connector) load(cur string) (map[string]entry, error) {
	h, ok := strings.CutPrefix(cur, "l1:")
	if !ok || len(h) != 24 || strings.Trim(h, "0123456789abcdef") != "" {
		return nil, fmt.Errorf("folder: %q isn't a folder cursor", cur)
	}
	var b []byte
	c.mu.Lock()
	if c.State == "" {
		b, ok = c.mem[cur]
		c.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("folder: the listing cursor %s names is gone", cur)
		}
	} else {
		c.mu.Unlock()
		var err error
		b, err = os.ReadFile(filepath.Join(c.State, h+".json"))
		if err != nil {
			return nil, fmt.Errorf("folder: the listing cursor %s names is gone: %w", cur, err)
		}
	}
	var l map[string]entry
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("folder: the listing %s can't be read: %w", cur, err)
	}
	return l, nil
}
