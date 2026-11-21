package laws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/backup/s3"
	"example.com/second-brain/internal/adapters/backup/s3/s3test"
)

// LAW-11 on a self-hosted cell (SB-1603): anchors and the off-site backup copy go to an
// S3 store with compliance-mode object lock. Nothing the cell can do overwrites or deletes
// an anchor inside its seven years, and the cell refuses a bucket whose lock is missing,
// in governance mode, shorter than seven years, or not enforced by the store.

func law11Box(r *rand.Rand) error {
	f := s3test.New("sb-worm", "sb-worm")
	srv := httptest.NewServer(f)
	defer srv.Close()
	creds, _ := json.Marshal(map[string]string{"access_key": "sb-worm", "secret_key": "s"})
	b := s3.Bucket{Endpoint: srv.URL, BucketName: "sb-worm", HTTP: srv.Client(), Label: "worm",
		Creds: func(context.Context) (string, error) { return string(creds), nil }}
	ctx := context.Background()
	if err := b.Check(ctx); err != nil {
		return fmt.Errorf("a seven-year compliance lock was refused: %v", err)
	}
	n := 1 + r.Intn(6)
	until := time.Now().AddDate(7, 0, 0)
	for i := 0; i < n; i++ {
		if err := b.Put(ctx, fmt.Sprintf("anchors/%012d.json", i), []byte(fmt.Sprintf("anchor %d", i)), until); err != nil {
			return err
		}
	}
	target := fmt.Sprintf("anchors/%012d.json", r.Intn(n))
	if r.Intn(2) == 0 {
		if err := b.Put(ctx, target, []byte("forged"), until); !errors.Is(err, s3.ErrExists) {
			return fmt.Errorf("anchor %s was overwritten: %v", target, err)
		}
	} else {
		req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/sb-worm/"+target+"?versionId=v1", nil)
		s3.Sign(req, nil, "sb-worm", "s", "us-east-1", time.Now())
		if resp, err := srv.Client().Do(req); err == nil {
			_ = resp.Body.Close()
		}
	}
	if got, err := b.Get(ctx, target); err != nil || !strings.HasPrefix(string(got), "anchor ") {
		return fmt.Errorf("anchor %s after the attempt: %q %v", target, got, err)
	}
	weak := []string{"", strings.Replace(s3test.Locked, "COMPLIANCE", "GOVERNANCE", 1),
		strings.Replace(s3test.Locked, "<Years>7</Years>", fmt.Sprintf("<Days>%d</Days>", 1+r.Intn(2554)), 1)}
	w := s3test.New("sb-worm", "sb-worm")
	w.Lock = weak[r.Intn(len(weak))]
	ws := httptest.NewServer(w)
	defer ws.Close()
	b.Endpoint, b.HTTP = ws.URL, ws.Client()
	if err := b.Check(ctx); !errors.Is(err, s3.ErrNotLocked) {
		return fmt.Errorf("a weak lock %q was accepted: %v", w.Lock, err)
	}
	u := s3test.New("sb-worm", "sb-worm")
	u.Unenforced = true
	us := httptest.NewServer(u)
	defer us.Close()
	b.Endpoint, b.HTTP = us.URL, us.Client()
	if err := b.Enforced(ctx); !errors.Is(err, s3.ErrNotLocked) {
		return fmt.Errorf("a store that lets a locked object be deleted was accepted: %v", err)
	}
	return nil
}
