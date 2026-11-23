package offsite

import (
	"bytes"
	"context"
	"crypto/rand"
	"time"

	"example.com/second-brain/internal/core/backup"
)

// Target seals every object before it reaches the copy it wraps, as NAME.sealed (sb-cell(5),
// backups[].seal; SB-1901): the nightly backup's off-box copy is unreadable without the
// recovery key, and the copies on the box stay as they are. Get and List see the sealed
// objects; the restore drill over this copy needs the recovery key (sb backup open).
type Target struct {
	backup.Target
	To PublicKey
}

// Put seals data and writes it as name.sealed.
func (t Target) Put(ctx context.Context, name string, data []byte, retainUntil time.Time) error {
	var b bytes.Buffer
	if err := Seal(&b, bytes.NewReader(data), t.To, rand.Reader); err != nil {
		return err
	}
	return t.Target.Put(ctx, name+".sealed", b.Bytes(), retainUntil)
}
