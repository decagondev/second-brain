package memory

import (
	"testing"

	"example.com/second-brain/internal/core/spacestest"
)

func TestContract(t *testing.T) { spacestest.Contract(t, New()) }
