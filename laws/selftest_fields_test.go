package laws

import (
	"math/rand"
	"testing"

	"example.com/second-brain/internal/adapters/connectors/mock"
	"example.com/second-brain/internal/core/ports"
)

func TestLAW07_StructuredFieldsOnlyWhenAllowed(t *testing.T) {
	Check(t, "LAW-07", runs, func(r *rand.Rand) error { return law07Fields(r, declareTruth) })
}

func TestLAW20_AllowedSensitiveFieldsAreRestricted(t *testing.T) {
	Check(t, "LAW-20", runs, func(r *rand.Rand) error { return law07Fields(r, declareTruth) })
}

// The fields law test catches a source that declares every field as not sensitive.
func TestSelf_FieldsMutationIsCaught(t *testing.T) {
	lying := func(src *mock.Source, truth []ports.Field) {
		var fs []ports.Field
		for _, f := range truth {
			fs = append(fs, ports.Field{Name: f.Name})
		}
		src.Declare(fs...)
	}
	if !violates(runs, func(r *rand.Rand) error { return law07Fields(r, lying) }) {
		t.Error("LAW-07 (fields): a source that hides which fields are sensitive was not caught")
	}
}
