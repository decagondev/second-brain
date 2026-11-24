package ingest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/fs"
	"net"

	"example.com/second-brain/internal/core/ports"
)

// The classes of a failed sync (SB-3201).
const (
	FailAuth        = "auth"
	FailRateLimited = "rate_limited"
	FailUnreachable = "unreachable"
	FailSource      = "source_error"
)

// FailureClass classes a sync's error by the status the source answered with, or by the
// network failure that kept it from answering. Anything else is source_error: never fine.
func FailureClass(err error) string {
	var se ports.StatusError
	if errors.As(err, &se) {
		switch se.HTTPStatus() {
		case 401, 403:
			return FailAuth
		case 429:
			return FailRateLimited
		default:
			return FailSource
		}
	}
	if errors.Is(err, ports.ErrRateLimited) {
		return FailRateLimited
	}
	var dns *net.DNSError
	var op *net.OpError
	var unk x509.UnknownAuthorityError
	var host x509.HostnameError
	var inv x509.CertificateInvalidError
	var rec tls.RecordHeaderError
	var ne net.Error
	switch {
	case errors.As(err, &dns), errors.As(err, &op), errors.As(err, &unk), errors.As(err, &host), errors.As(err, &inv), errors.As(err, &rec),
		errors.Is(err, context.DeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		return FailUnreachable
	}
	return FailSource
}

// KnownClass is class when it is one of the four, else source_error: a record written by
// another version, or none, is never read as fine.
func KnownClass(class string) string {
	switch class {
	case FailAuth, FailRateLimited, FailUnreachable, FailSource:
		return class
	}
	return FailSource
}

// Gone reports whether a fetch failed because the source no longer has the document: a 404
// or 410, or a missing file. Any other failure is not gone.
func Gone(err error) bool {
	var se ports.StatusError
	if errors.As(err, &se) {
		return se.HTTPStatus() == 404 || se.HTTPStatus() == 410
	}
	return errors.Is(err, fs.ErrNotExist)
}
