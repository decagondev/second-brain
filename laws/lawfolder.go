package laws

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"example.com/second-brain/internal/adapters/connectors/folder"
	"example.com/second-brain/internal/adapters/embed/hashing"
	memindex "example.com/second-brain/internal/adapters/index/memory"
	"example.com/second-brain/internal/core/ingest"
	"example.com/second-brain/internal/core/ports"
)

// The same connector laws over the folder connector (SB-1606), on a real directory.
// LAW-03: an ACL file that can't be read, a share slower than the deadline, or ACL files
// that are gone are never Allow; a "script" fault is a document with no ACL file, which is
// Unknown, never allowed. LAW-13: the index built from the share is the index rebuilt.

// faultFS fails every ACL file read while a fault is set.
type faultFS struct {
	fs.FS
	mu    sync.Mutex
	fault string
}

func (f *faultFS) set(s string) { f.mu.Lock(); f.fault = s; f.mu.Unlock() }

func (f *faultFS) Open(name string) (fs.File, error) {
	f.mu.Lock()
	fault := f.fault
	f.mu.Unlock()
	if strings.HasSuffix(name, ".acl") {
		switch fault {
		case "500":
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
		case "timeout":
			time.Sleep(folder.ACLDeadline + 100*time.Millisecond)
		case "404":
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
	}
	return f.FS.Open(name)
}

func openFolder(dir string) (*folder.Connector, *faultFS, error) {
	c, err := folder.Open(dir, filepath.Join(dir, ".state"))
	if err != nil {
		return nil, nil, err
	}
	f := &faultFS{FS: c.FS}
	c.FS = f
	return c, f, nil
}

// NewFolderWorld builds the connector-level world over a share in dir.
var NewFolderWorld = func(dir string) ConnectorWorld {
	c, f, err := openFolder(dir)
	if err != nil {
		panic(err)
	}
	return &folderWorld{dir: dir, c: c, f: f}
}

type folderWorld struct {
	dir string
	c   *folder.Connector
	f   *faultFS
	n   int
}

func (w *folderWorld) put(id, text string, readers []string) error {
	p := filepath.Join(w.dir, filepath.FromSlash(id))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		return err
	}
	if readers == nil {
		return nil
	}
	var acl strings.Builder
	for _, r := range readers {
		if strings.Contains(r, "@") {
			fmt.Fprintf(&acl, "read %s\n", r)
		}
	}
	return os.WriteFile(p+".acl", []byte(acl.String()), 0o600)
}

func (w *folderWorld) ACL(access []bool, fault string) ([]string, error) {
	w.n++
	refs := make([]ports.SourceRef, len(access))
	for i, ok := range access {
		id := fmt.Sprintf("b%d/d%d.md", w.n, i)
		reader := "bob@acme.example"
		if ok {
			reader = "ann@acme.example"
		}
		readers := []string{reader}
		if fault == "script" && i == 0 {
			readers = nil // no ACL file at all
		}
		if err := w.put(id, "text", readers); err != nil {
			return nil, err
		}
		refs[i] = ports.SourceRef{Kind: "docs", ID: id}
	}
	if fault != "script" {
		w.f.set(fault)
		defer w.f.set("")
	}
	vs, err := w.c.ACL(context.Background(), ports.UserRef{ID: "usr_ann", Email: "ann@acme.example"}, refs)
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out, err
}

func (w *folderWorld) Act(string, bool) (string, error) {
	return "", ErrPending // read-only: LAW-05 doesn't apply
}

func (w *folderWorld) Effects() int { return 0 }

// NewFolderIngestWorld runs LAW-13 over the folder connector on a share in dir.
var NewFolderIngestWorld = func(dir string) IngestWorld {
	c, f, err := openFolder(dir)
	if err != nil {
		panic(err)
	}
	idx := memindex.New()
	return &folderIngest{w: &folderWorld{dir: dir, c: c, f: f}, ingestWorld: ingestWorld{idx: idx, p: ingest.Pipeline{Tenant: "acme-logistics",
		Release: "rel_test", Connector: c, Embedder: hashing.Embedder{}, Index: idx}}}
}

type folderIngest struct {
	w *folderWorld
	ingestWorld
	edits int
}

func (w *folderIngest) Load(docs []SourceDoc) error {
	for _, d := range docs {
		if err := w.w.put(d.ID+".md", d.Text, append([]string{}, d.Viewers...)); err != nil {
			return err
		}
	}
	return nil
}

func (w *folderIngest) Edit(id, text string) error {
	p := filepath.Join(w.w.dir, filepath.FromSlash(id)+".md")
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		return err
	}
	w.edits++
	at := time.Now().Add(time.Duration(w.edits) * time.Minute) // a new version, even within the clock's resolution
	return os.Chtimes(p, at, at)
}
