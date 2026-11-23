package wire

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/core/notify"
)

// SB-2607: the notify job's last run is kept private to the cell, counts only, and read back.
func TestNotifyRunIsKept(t *testing.T) {
	c := &Cell{Config: Config{State: t.TempDir()}}
	c.Config.Mattermost.Base = "https://mm.acme.example"
	if _, found, err := lastNotify(c.Config.State); found || err != nil {
		t.Fatalf("before any run: %v %v", found, err)
	}
	at := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)
	if err := c.RecordNotify(notify.Result{Sent: 2, Failed: 1}, errors.New("ann@acme.example: refused"), at); err != nil {
		t.Fatal(err)
	}
	rec, found, err := lastNotify(c.Config.State)
	if err != nil || !found || rec.Via != "mattermost" || rec.Sent != 2 || rec.Failed != 1 || !rec.Error || !rec.At.Equal(at) {
		t.Fatalf("%+v %v %v", rec, found, err)
	}
	st, err := os.Stat(notifyRun(c.Config.State))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("mode: %v %v", st, err)
	}
	b, _ := os.ReadFile(notifyRun(c.Config.State))
	if strings.Contains(string(b), "@") {
		t.Fatalf("the record says who: %s", b)
	}
}
