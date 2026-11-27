// Package record turns what a real sandbox answered into a recording a fake can be checked
// against (SB-6002; sb(1), sb record): every string value is replaced by one of the same
// shape and length, the same value always by the same replacement, and every address by a
// person at one of the placeholder companies. Keys, numbers and booleans stay. Shape is a
// body's structure, and Missing names the fields a fake's body lacks that a recording has.
package record

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Exchange is one step of a recording script and what it answered.
type Exchange struct {
	Step   string          `json:"step"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// Recording is a vendor's recorded exchanges.
type Recording struct {
	Vendor    string     `json:"vendor"`
	API       string     `json:"api"`
	Recorded  string     `json:"recorded"` // the day, YYYY-MM-DD
	Exchanges []Exchange `json:"exchanges"`
}

// Companies are the placeholder companies people are put in.
var Companies = []string{"acme-logistics", "beta-freight", "gamma-parts"}

// Scrubber replaces values consistently within one recording.
type Scrubber struct {
	seen   map[string]string
	people int
	r      *rand.Rand
}

// NewScrubber starts a scrubber; seed only makes the replacements repeatable in tests.
func NewScrubber(seed int64) *Scrubber {
	return &Scrubber{seen: map[string]string{}, r: rand.New(rand.NewSource(seed))}
}

var emailRe = regexp.MustCompile(`[A-Za-z0-9._%+'-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)

// keep are values the vendor's protocol fixes, kept as they are.
var keep = regexp.MustCompile(`^(#?microsoft\.graph\.[A-Za-z.]+|[a-z]+/[a-z0-9.+-]+|text|html|deleted|anonymous|organization|users|view|edit|read|write|owner|true|false|)$`)

// Value is s replaced: an address becomes a person at a placeholder company, a value the
// protocol fixes stays, and anything else keeps its shape (letters for letters, digits for
// digits, everything else as it is) and length.
func (s *Scrubber) Value(v string) string {
	if keep.MatchString(v) {
		return v
	}
	if out, ok := s.seen[v]; ok {
		return out
	}
	var out string
	if emailRe.MatchString(v) {
		out = emailRe.ReplaceAllStringFunc(v, func(a string) string {
			if r, ok := s.seen[a]; ok {
				return r
			}
			s.people++
			r := fmt.Sprintf("person%d@%s.example", s.people, Companies[(s.people-1)%len(Companies)])
			s.seen[a] = r
			return r
		})
		out = s.shape(emailRe.Split(out, -1), emailRe.FindAllString(out, -1))
	} else {
		out = s.shape([]string{v}, nil)
	}
	s.seen[v] = out
	return out
}

// shape replaces the characters of parts, keeping the kept pieces between them.
func (s *Scrubber) shape(parts, kept []string) string {
	var b strings.Builder
	for i, p := range parts {
		for _, c := range p {
			switch {
			case c >= 'a' && c <= 'z':
				b.WriteRune(rune('a' + s.r.Intn(26)))
			case c >= 'A' && c <= 'Z':
				b.WriteRune(rune('A' + s.r.Intn(26)))
			case c >= '0' && c <= '9':
				b.WriteRune(rune('0' + s.r.Intn(10)))
			case c > 127:
				b.WriteRune('x')
			default:
				b.WriteRune(c)
			}
		}
		if i < len(kept) {
			b.WriteString(kept[i])
		}
	}
	return b.String()
}

// Body scrubs every string in a JSON body; a body that isn't JSON is dropped.
func (s *Scrubber) Body(raw []byte) json.RawMessage {
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if len(bytes.TrimSpace(raw)) == 0 || d.Decode(&v) != nil {
		return nil
	}
	out, _ := json.Marshal(s.walk(v))
	return out
}

func (s *Scrubber) walk(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, x := range t {
			out[k] = s.walk(x)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = s.walk(x)
		}
		return out
	case string:
		return s.Value(t)
	}
	return v
}

// New is a recording of vendor's exchanges, each body scrubbed.
func New(vendor, api string, now time.Time, ex []Exchange, s *Scrubber) Recording {
	rec := Recording{Vendor: vendor, API: api, Recorded: now.UTC().Format("2006-01-02")}
	for _, e := range ex {
		rec.Exchanges = append(rec.Exchanges, Exchange{Step: e.Step, Status: e.Status, Body: s.Body(e.Body)})
	}
	return rec
}

// ErrReal is a recording with something in it that looks real.
var ErrReal = errors.New("record: holds a real-looking value")

var allowedMail = regexp.MustCompile(`^person[0-9]+@(acme-logistics|beta-freight|gamma-parts)\.example$`)

// CheckPeople refuses a recording with an address that isn't a placeholder person.
func CheckPeople(b []byte) error {
	for _, a := range emailRe.FindAllString(string(b), -1) {
		if !allowedMail.MatchString(a) {
			return fmt.Errorf("%w: an address", ErrReal)
		}
	}
	return nil
}

// Shape is a body's fields as paths, with each one's JSON type: "value[].id" string.
func Shape(raw []byte) map[string]string {
	out := map[string]string{}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return out
	}
	var walk func(p string, v any)
	walk = func(p string, v any) {
		switch t := v.(type) {
		case map[string]any:
			if p != "" {
				out[p] = "object"
			}
			for k, x := range t {
				q := k
				if p != "" {
					q = p + "." + k
				}
				walk(q, x)
			}
		case []any:
			out[p] = "array"
			for _, x := range t {
				walk(p+"[]", x)
			}
		case string:
			out[p] = "string"
		case float64:
			out[p] = "number"
		case bool:
			out[p] = "boolean"
		case nil:
			out[p] = "null"
		}
	}
	walk("", v)
	return out
}

// Missing are the fields of want (a recording's) that got (a fake's) lacks or types
// differently, sorted. A null in either matches anything.
func Missing(want, got map[string]string) []string {
	var out []string
	for p, t := range want {
		g, ok := got[p]
		switch {
		case !ok:
			out = append(out, p)
		case g != t && g != "null" && t != "null":
			out = append(out, p+" ("+g+", not "+t+")")
		}
	}
	sort.Strings(out)
	return out
}
