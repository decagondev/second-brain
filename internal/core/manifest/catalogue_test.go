package manifest

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// SB-1101: catalogue.json and the table in sb-departments(7) say the same thing, so
// neither can change alone.
func TestCatalogueMatchesTheManual(t *testing.T) {
	page, err := os.ReadFile("../../../manual/sb-departments.7.md")
	if err != nil {
		t.Fatal(err)
	}
	row := regexp.MustCompile("(?m)^\\| `([a-z-]+)` \\| ([a-z]+) \\| ([^|]+) \\| ([^|]+) \\| ([^|]+) \\|")
	seen := 0
	for _, m := range row.FindAllStringSubmatch(string(page), -1) {
		tpl, ok := Templates[m[1]]
		if !ok {
			t.Errorf("the manual lists %s, the catalogue doesn't", m[1])
			continue
		}
		seen++
		sources := []string{}
		if s := strings.TrimSpace(m[3]); s != "any" {
			for _, x := range strings.Split(s, ",") {
				sources = append(sources, strings.TrimSpace(x))
			}
		}
		agents := regexp.MustCompile("`([a-z-]+)`").FindAllStringSubmatch(m[4], -1)
		var names []string
		for _, a := range agents {
			names = append(names, a[1])
		}
		ret := 0
		if strings.HasPrefix(strings.TrimSpace(m[5]), "365") {
			ret = 365
		}
		if tpl.Label != m[2] || !reflect.DeepEqual(tpl.Sources, sources) || !reflect.DeepEqual(tpl.Agents, names) || tpl.RetentionDays != ret {
			t.Errorf("%s: catalogue %+v, manual label %s sources %v agents %v retention %d", m[1], tpl, m[2], sources, names, ret)
		}
	}
	if seen != len(Templates) || seen != 13 {
		t.Fatalf("the manual's table has %d of the catalogue's %d templates; want 13", seen, len(Templates))
	}
}

func TestParseCatalogueRefuses(t *testing.T) {
	ok := `{"version":1,"templates":[{"name":"custom","label":"internal","sources":[],"agents":["ask"],"retention_days":0,"suite":""}]}`
	if _, err := ParseCatalogue([]byte(ok)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(ok, `"version":1`, `"version":2`, 1),
		strings.Replace(ok, `"internal"`, `"secret"`, 1),
		strings.Replace(ok, `["ask"]`, `["draft-reply"]`, 1),
		strings.Replace(ok, `"retention_days":0`, `"retention_days":7`, 1),
		strings.Replace(ok, `"custom"`, `"finance"`, 1),
		strings.Replace(ok, `"suite":""`, `"suite":"","extra":1`, 1),
		strings.Replace(ok, `]}`, `,{"name":"custom","label":"internal","sources":[],"agents":["ask"],"retention_days":0,"suite":""}]}`, 1),
	} {
		if _, err := ParseCatalogue([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}
