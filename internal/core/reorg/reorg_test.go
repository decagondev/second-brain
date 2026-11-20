package reorg

import (
	"errors"
	"testing"
)

func TestParseRefuses(t *testing.T) {
	for name, src := range map[string]string{
		"version":       `{"version": 2, "people": []}`,
		"unknown field": `{"version": 1, "people": [], "x": 1}`,
		"twice":         `{"version": 1, "people": [{"user": "a@x.example", "before": [], "after": []}, {"user": "A@x.example", "before": [], "after": []}]}`,
		"no user":       `{"version": 1, "people": [{"user": " ", "before": [], "after": []}]}`,
		"not a group":   `{"version": 1, "people": [{"user": "a@x.example", "before": ["finance"], "after": []}]}`,
		"two values":    `{"version": 1, "people": []} {}`,
	} {
		if _, err := Parse([]byte(src)); !errors.Is(err, ErrSnapshot) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
