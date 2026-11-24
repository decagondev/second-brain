package ingest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"testing"
)

type status int

func (s status) Error() string   { return fmt.Sprintf("HTTP %d", int(s)) }
func (s status) HTTPStatus() int { return int(s) }

// SB-3201: a failed sync is classed by what the source answered, or didn't; anything the
// cell can't place is source_error, never fine.
func TestFailureClass(t *testing.T) {
	for err, want := range map[error]string{
		fmt.Errorf("github: %w", status(401)):                   FailAuth,
		fmt.Errorf("x: %w", status(403)):                        FailAuth,
		fmt.Errorf("x: %w", status(429)):                        FailRateLimited,
		fmt.Errorf("x: %w", status(502)):                        FailSource,
		fmt.Errorf("x: %w", &net.DNSError{Err: "no such host"}): FailUnreachable,
		fmt.Errorf("x: %w", &net.OpError{Op: "dial", Err: nil}): FailUnreachable,
		fmt.Errorf("x: %w", context.DeadlineExceeded):           FailUnreachable,
		errors.New("something else"):                            FailSource,
	} {
		if got := FailureClass(err); got != want {
			t.Errorf("%v: %s, want %s", err, got, want)
		}
	}
	if KnownClass("") != FailSource || KnownClass("weird") != FailSource || KnownClass(FailAuth) != FailAuth {
		t.Fatal("an unknown class read as something other than source_error")
	}
	if !Gone(fmt.Errorf("x: %w", status(404))) || !Gone(fmt.Errorf("folder: %w", fs.ErrNotExist)) || Gone(fmt.Errorf("x: %w", status(500))) || Gone(errors.New("x")) {
		t.Fatal("gone")
	}
}
