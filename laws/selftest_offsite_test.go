package laws

import (
	"context"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/second-brain/internal/adapters/backup/s3"
)

// The bucket law catches a push that sends whatever it is given (SB-2303).
type pushAnything struct{}

func (pushAnything) Push(file, to, keys string) error {
	u, _ := url.Parse(to)
	data, _ := os.ReadFile(file)
	kb, _ := os.ReadFile(keys)
	ep := strings.SplitN(strings.SplitN(string(kb), `"endpoint":"`, 2)[1], `"`, 2)[0]
	b := s3.Bucket{Endpoint: ep, BucketName: u.Host, Prefix: strings.Trim(u.Path, "/"), Plain: true,
		Creds: func(context.Context) (string, error) { return string(kb), nil }}
	return b.Put(context.Background(), "db/"+filepath.Base(file), data, time.Time{})
}

func TestSelfOffsitePushAnythingIsCaught(t *testing.T) {
	dir := t.TempDir()
	failsSomewhere(t, "unsealed push", func(r *rand.Rand) error { return law07Offsite(r, pushAnything{}, dir) })
}
