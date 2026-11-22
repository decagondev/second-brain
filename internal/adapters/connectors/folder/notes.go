package folder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"example.com/second-brain/internal/core/ports"
)

// Notes (sb-connector(3), docs: folder, Notes; SB-1705): the one thing the connector
// writes, a steward's note and its ACL file under notes/, through the tool bus.

// NotesDir is the folder of the share notes are written to.
const NotesDir = "notes"

// MaxNote is the largest note, in bytes.
const MaxNote = 64 << 10

var notePath = regexp.MustCompile(`^notes/([a-z][a-z0-9-]{1,30}|company)/qs_[A-Za-z0-9_]{1,40}\.md$`)

// EnableNotes opens dir/notes as a root of its own for writes, making it if it isn't there.
// Nothing else of the share is ever opened for writing.
func (c *Connector) EnableNotes(dir string) error {
	p := filepath.Join(dir, NotesDir)
	_ = os.Mkdir(p, 0o755) // a read-only share must already have it (sb-selfhost(7))
	r, err := os.OpenRoot(p)
	if err != nil {
		return fmt.Errorf("folder: notes: %w", err)
	}
	c.notes = r
	return nil
}

type notePayload struct {
	Markdown string   `json:"markdown"`
	Readers  []string `json:"readers"`
	Owner    string   `json:"owner"`
	Label    string   `json:"label"`
}

// Act implements ports.Connector: only note-publish's write of a note, and only with notes
// enabled. Anything else is ENOTSUP, as for a read-only connector.
func (c *Connector) Act(_ context.Context, a ports.Action, _ ports.IdempotencyKey) (ports.Receipt, error) {
	if c.notes == nil {
		return ports.Receipt{}, errors.New("ENOTSUP: folder connector is read-only")
	}
	if !notePath.MatchString(a.Target) {
		return ports.Receipt{}, fmt.Errorf("EINVAL: a note is written only to notes/SPACE/qs_ID.md")
	}
	var p notePayload
	d := json.NewDecoder(bytes.NewReader(a.Payload))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil || strings.TrimSpace(p.Markdown) == "" || len(p.Markdown) > MaxNote {
		return ports.Receipt{}, errors.New("EINVAL: a note's payload is {markdown, readers, owner, label}")
	}
	acl := "owner " + p.Owner + "\n"
	for _, r := range p.Readers {
		acl += "read " + r + "\n"
	}
	if p.Label != "" {
		acl += "label " + p.Label + "\n"
	}
	if _, err := parseACL([]byte(acl)); err != nil || len(p.Readers) == 0 || strings.ContainsAny(p.Owner+strings.Join(p.Readers, ""), "\n\r") {
		return ports.Receipt{}, fmt.Errorf("EINVAL: a note's readers and owner must be emails, its label [a-z-]")
	}
	rel := strings.TrimPrefix(a.Target, NotesDir+"/")
	if err := c.notes.Mkdir(filepath.Dir(rel), 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return ports.Receipt{}, fmt.Errorf("folder: notes: %w", err)
	}
	// The note first: until its ACL file is there it is Unknown to everyone (LAW-03).
	if err := c.create(rel, []byte(p.Markdown)); err != nil {
		return ports.Receipt{}, err
	}
	if err := c.create(rel+".acl", []byte(acl)); err != nil {
		return ports.Receipt{}, err
	}
	back, err := c.readNote(rel)
	if err != nil || !bytes.Equal(back, []byte(p.Markdown)) {
		return ports.Receipt{}, fmt.Errorf("folder: the note doesn't read back as written")
	}
	if b, err := c.readNote(rel + ".acl"); err != nil || string(b) != acl {
		return ports.Receipt{}, fmt.Errorf("folder: the note's ACL file doesn't read back as written")
	}
	sum := sha256.Sum256(back)
	return ports.Receipt{TargetID: a.Target + "@sha256:" + hex.EncodeToString(sum[:])}, nil
}

// create writes a new file under notes/, never over another: the same bytes again are
// the same note (an idempotent retry), different ones are an error.
func (c *Connector) create(rel string, b []byte) error {
	f, err := c.notes.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		have, rerr := c.readNote(rel)
		if rerr == nil && bytes.Equal(have, b) {
			return nil
		}
		return fmt.Errorf("folder: %s is already a different note; notes are never overwritten", rel)
	}
	if err != nil {
		return fmt.Errorf("folder: notes: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return fmt.Errorf("folder: notes: %w", err)
	}
	return f.Close()
}

func (c *Connector) readNote(rel string) ([]byte, error) {
	f, err := c.notes.Open(rel)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, MaxNote+1))
}
