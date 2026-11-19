// Package version reports the build version. Values are set at link time:
//
//	go build -ldflags "-X example.com/second-brain/internal/version.Version=1.2.3 -X example.com/second-brain/internal/version.Commit=abc123" ./cmd/sb
package version

var (
	Version = "0.0.0-dev"
	Commit  = "unknown"
)
