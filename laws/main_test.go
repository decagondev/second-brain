package laws

import (
	"os"
	"testing"
)

// TestMain removes the release stores the worlds created (BUG-0013).
func TestMain(m *testing.M) {
	code := m.Run()
	tempDirs.RemoveAll()
	os.Exit(code)
}
