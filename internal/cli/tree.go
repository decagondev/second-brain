package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/reorg"
	"example.com/second-brain/internal/errs"
)

// treePlan is the reorg dry run (sb-space(1), tree; SB-4806): sb tree move FILE SPACE --to
// PARENT --plan, or sb tree merge FILE FROM INTO --plan. It reads the manifest offline and
// prints what each affected space would inherit or lose and where its questions would
// climb. Nothing moves: --write OUT saves the manifest the change would leave, for review.
func (e *env) treePlan(op string, args []string) int {
	name := "sb tree " + op
	usage := errs.New(errs.EUSAGE, "usage: sb tree move FILE SPACE --to PARENT --plan [--write OUT] | sb tree merge FILE FROM INTO --plan [--write OUT]")
	args = e.localArgs(args)
	var pos []string
	var to, out string
	var dry, hasTo bool
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--plan":
			dry = true
		case a == "--to" && i+1 < len(args) && op == reorg.OpMove:
			to, hasTo = args[i+1], true
			i++
		case a == "--write" && i+1 < len(args):
			out = args[i+1]
			i++
		case !strings.HasPrefix(a, "--"):
			pos = append(pos, a)
		default:
			return e.fail(name, usage)
		}
	}
	want := 3
	if op == reorg.OpMove {
		want = 2
	}
	if len(pos) != want || (op == reorg.OpMove && !hasTo) {
		return e.fail(name, usage)
	}
	if !dry {
		return e.fail(name, errs.New(errs.EUSAGE, "only --plan: a tree change is a manifest diff that goes through review"))
	}
	t, code, ok := e.readManifest(name, pos[0])
	if !ok {
		return code
	}
	var p reorg.TreePlan
	var err error
	if op == reorg.OpMove {
		p, err = reorg.Move(t, pos[1], to)
	} else {
		p, err = reorg.Merge(t, pos[1], pos[2])
	}
	switch {
	case errors.Is(err, reorg.ErrWiden):
		return e.fail(name, errs.New(errs.EPERM, "%v", err))
	case err != nil:
		return e.fail(name, errs.New(errs.EINVAL, "%v", err))
	}
	if out != "" {
		if err := os.WriteFile(out, manifest.Format(p.After), 0o644); err != nil {
			return e.fail(name, errs.New(errs.EUSAGE, "cannot write %s", out))
		}
	}
	if e.g.JSON {
		return e.json(p)
	}
	e.printTreePlan(p)
	return 0
}

func (e *env) printTreePlan(p reorg.TreePlan) {
	dash := func(xs []string) string {
		if len(xs) == 0 {
			return "-"
		}
		return strings.Join(xs, " > ")
	}
	for _, s := range p.Spaces {
		if s.Gone {
			fmt.Fprintf(e.out, "%s  merged into %s\n", s.Space, p.To)
			continue
		}
		fmt.Fprintf(e.out, "%s  parent %s -> %s  escalates %s -> %s\n", s.Space, orDash(s.ParentBefore), orDash(s.ParentAfter), dash(s.EscalateNow), dash(s.EscalateAfter))
		for _, c := range s.Settings {
			fmt.Fprintf(e.out, "  %s  gains %s  loses %s\n", c.Setting, orDash(strings.Join(c.Gains, ",")), orDash(strings.Join(c.Loses, ",")))
		}
	}
	fmt.Fprintln(e.out, "nothing applied; propose the manifest diff for review")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
