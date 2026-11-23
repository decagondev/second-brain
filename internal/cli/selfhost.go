package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"example.com/second-brain/internal/core/box"
	"example.com/second-brain/internal/errs"
)

// selfhostDoctor prints the verdict of every check over the facts the host step gathered
// (sb-selfhost(7), Doctor; SB-1801): `sb selfhost doctor [--facts FILE]`, stdin by default.
func (e *env) selfhostDoctor(args []string) int {
	path := "-"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--facts":
			if i+1 >= len(args) {
				return e.fail("sb selfhost doctor", errs.New(errs.EUSAGE, "--facts needs a file"))
			}
			path = args[i+1]
			i++
		default:
			return e.fail("sb selfhost doctor", errs.New(errs.EUSAGE, "unknown argument %q", args[i]))
		}
	}
	var b []byte
	var err error
	if path == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return e.fail("sb selfhost doctor", errs.New(errs.EINVAL, "can't read the facts: %v", err))
	}
	f, err := box.Parse(b)
	if err != nil {
		return e.fail("sb selfhost doctor", errs.New(errs.EINVAL, "%v", err))
	}
	lines := box.Check(f, time.Now())
	code := box.ExitCode(lines)
	if e.g.JSON {
		e.json(map[string]any{"verdict": box.Worst(lines), "checks": lines})
		return code
	}
	for _, l := range lines {
		fmt.Fprintf(e.out, "%-7s %-32s %s\n", l.Verdict, l.Check, l.Detail)
		if l.Fix != "" {
			fmt.Fprintf(e.out, "        fix: %s\n", l.Fix)
		}
	}
	fmt.Fprintf(e.out, "doctor: %s\n", box.Worst(lines))
	return code
}
