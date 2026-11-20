package cli

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/reorg"
	"example.com/second-brain/internal/errs"
)

// spaceTemplates prints the catalogue this release ships (sb-space(1) templates).
func (e *env) spaceTemplates(args []string) int {
	const name = "sb space templates"
	if len(e.localArgs(args)) != 0 {
		return e.fail(name, errs.New(errs.EUSAGE, "usage: sb space templates"))
	}
	names := make([]string, 0, len(manifest.Templates))
	for n := range manifest.Templates {
		names = append(names, n)
	}
	sort.Strings(names)
	if e.g.JSON {
		out := make([]manifest.Template, 0, len(names))
		for _, n := range names {
			out = append(out, manifest.Templates[n])
		}
		return e.json(map[string]any{"templates": out})
	}
	for _, n := range names {
		t := manifest.Templates[n]
		src := strings.Join(t.Sources, ",")
		if src == "" {
			src = "any"
		}
		fmt.Fprintf(e.out, "%-12s %-12s sources %-26s agents %s\n", n, t.Label, src, strings.Join(t.Agents, ","))
	}
	return 0
}

var spacesKey = regexp.MustCompile(`(?m)^spaces:[ \t]*(#.*)?$`)

// spaceInit adds one space from a template to a manifest file (sb-space(1) init). The file
// is written only if the result passes every rule of sb tenant check.
func (e *env) spaceInit(args []string) int {
	const name = "sb space init"
	usage := errs.New(errs.EUSAGE, "usage: sb space init FILE --template NAME --members GROUP[,GROUP] [--name NAME] [--stewards GROUP[,GROUP]]")
	args = e.localArgs(args)
	var file, tpl, spName string
	var members, stewards []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			if file != "" {
				return e.fail(name, usage)
			}
			file = a
			continue
		}
		if i+1 >= len(args) {
			return e.fail(name, errs.New(errs.EUSAGE, "%s needs a value", a))
		}
		v := args[i+1]
		i++
		switch a {
		case "--template":
			tpl = v
		case "--name":
			spName = v
		case "--members":
			members = strings.Split(v, ",")
		case "--stewards":
			stewards = strings.Split(v, ",")
		default:
			return e.fail(name, usage)
		}
	}
	if file == "" || tpl == "" || len(members) == 0 {
		return e.fail(name, usage)
	}
	t, ok := manifest.Templates[tpl]
	if !ok {
		return e.fail(name, errs.New(errs.EUSAGE, "no template %q; see sb space templates", tpl))
	}
	if spName == "" {
		spName = tpl
	}
	if t.Label == "restricted" && len(stewards) == 0 {
		return e.fail(name, errs.New(errs.EUSAGE, "template %s is restricted, so the space needs --stewards", tpl))
	}
	src, err := os.ReadFile(file)
	if err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "cannot read %s", file))
	}
	before, code, ok := e.readManifest(name, file)
	if !ok {
		return code
	}
	if _, exists := before.Spaces[spName]; exists {
		return e.fail(name, errs.New(errs.EMANIFEST, "%s already has a space %q", file, spName))
	}
	scope := map[string][]string{}
	for _, kind := range t.Sources {
		if _, has := before.Connectors[kind]; has {
			scope[kind] = []string{defaultPattern(kind, spName)}
		}
	}
	if len(scope) == 0 {
		need := strings.Join(t.Sources, " or ")
		if need == "" {
			need = "any kind"
		}
		return e.fail(name, errs.New(errs.EMANIFEST, "%s has no connector for template %s's sources; add a connector (%s) first", file, tpl, need))
	}
	entry := spaceEntry(spName, tpl, members, stewards, scope)
	text := string(src)
	switch loc := spacesKey.FindStringIndex(text); {
	case loc != nil:
		end := loc[1]
		if end < len(text) && text[end] == '\n' {
			end++
		} else {
			text = text[:end] + "\n" + text[end:]
			end++
		}
		text = text[:end] + entry + text[end:]
	case before.Spaces != nil:
		return e.fail(name, errs.New(errs.EMANIFEST, "%s writes spaces: in flow style; add the space by hand", file))
	default:
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "spaces:\n" + entry
	}
	after, ds := manifest.Check([]byte(text))
	if len(ds) > 0 {
		for _, d := range ds {
			fmt.Fprintln(e.out, d.Format(file))
		}
		return e.fail(name, errs.New(errs.EMANIFEST, "the new space doesn't check; %s is unchanged", file))
	}
	if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
		return e.fail(name, errs.New(errs.EUNAVAIL, "cannot write %s", file))
	}
	if e.g.JSON {
		return e.json(map[string]any{"file": file, "space": spName, "added": after.Spaces[spName]})
	}
	fmt.Fprintf(e.out, "space %s added to %s (template %s, label %s)\n", spName, file, tpl, after.Spaces[spName].Label)
	return 0
}

// defaultPattern is a space's starting scope for one connector kind (sb-space(1)).
func defaultPattern(kind, space string) string {
	switch kind {
	case "docs":
		return "/" + strings.ToUpper(space[:1]) + space[1:] + "/**"
	case "chat":
		return "#" + space + "-*"
	case "tickets":
		return strings.ToUpper(strings.ReplaceAll(space, "-", ""))
	}
	return "*"
}

func spaceEntry(name, tpl string, members, stewards []string, scope map[string][]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %s:\n    template: %s\n    members: [%s]\n", name, tpl, strings.Join(members, ", "))
	if len(stewards) > 0 {
		fmt.Fprintf(&b, "    stewards: [%s]\n", strings.Join(stewards, ", "))
	}
	kinds := make([]string, 0, len(scope))
	for k := range scope {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	b.WriteString("    scope:\n")
	for _, k := range kinds {
		fmt.Fprintf(&b, "      %s: [%q]\n", k, scope[k][0])
	}
	return b.String()
}

// spaceDiff previews a reorg's effect on spaces (sb-space(1) diff; SB-1206). It reads the
// manifest and the snapshot and writes nothing.
func (e *env) spaceDiff(args []string) int {
	const name = "sb space diff"
	usage := errs.New(errs.EUSAGE, "usage: sb space diff FILE --idp-snapshot SNAPSHOT")
	args = e.localArgs(args)
	var file, snapFile string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--idp-snapshot" && i+1 < len(args):
			snapFile = args[i+1]
			i++
		case !strings.HasPrefix(a, "--") && file == "":
			file = a
		default:
			return e.fail(name, usage)
		}
	}
	if file == "" || snapFile == "" {
		return e.fail(name, usage)
	}
	t, code, ok := e.readManifest(name, file)
	if !ok {
		return code
	}
	b, err := os.ReadFile(snapFile)
	if err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "cannot read %s", snapFile))
	}
	snap, err := reorg.Parse(b)
	if err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "%s: %v", snapFile, err))
	}
	r := reorg.Diff(t, snap)
	if e.g.JSON {
		return e.json(r)
	}
	list := func(xs []string) string {
		if len(xs) == 0 {
			return "-"
		}
		return strings.Join(xs, ",")
	}
	for _, c := range r.People {
		line := fmt.Sprintf("%s  gains %s  loses %s", c.User, list(c.Gains), list(c.Loses))
		if len(c.Flags) > 0 {
			line += "  [" + strings.Join(c.Flags, ",") + "]"
		}
		fmt.Fprintln(e.out, line)
	}
	fmt.Fprintf(e.out, "%d unchanged\n", r.Unchanged)
	return 0
}
