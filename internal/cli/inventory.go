package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"example.com/second-brain/internal/core/inventory"
	"example.com/second-brain/internal/errs"
)

// inventoryExport is sb inventory export [--tenant T | --cell URL] [--out DIR] [--key KEY]
// (sb-admin(3), Inventory; SB-5007): the running cell's signed AI inventory, checked before
// it writes ai-inventory.json and ai-inventory.csv.
func (e *env) inventoryExport(args []string) int {
	const name = "sb inventory export"
	const usage = "usage: sb inventory export [--tenant T | --cell URL] [--out DIR] [--key ed25519:PUB]"
	var o cellOpts
	out, key := ".", ""
	args = e.localArgs(args)
	for i := 0; i < len(args); i++ {
		if took, err := o.take(args, &i); took {
			if err != nil {
				return e.fail(name, err)
			}
			continue
		}
		switch a := args[i]; a {
		case "--out", "--key":
			if i+1 >= len(args) {
				return e.fail(name, errs.New(errs.EUSAGE, "%s needs a value", a))
			}
			if a == "--out" {
				out = args[i+1]
			} else {
				key = args[i+1]
			}
			i++
		default:
			return e.fail(name, errs.New(errs.EUSAGE, usage))
		}
	}
	cellURL, _, err := e.cellURL(o)
	if err != nil {
		return e.fail(name, err)
	}
	raw, err := cellDo(http.MethodGet, cellURL, "/v1/cell/inventory", "", nil, 2*time.Minute)
	if err != nil {
		return e.fail(name, err)
	}
	inv, err := inventory.Verify(raw, key)
	if err != nil {
		return e.fail(name, errs.New(errs.ECHAIN, "the inventory doesn't verify: %v", err))
	}
	js, csv := filepath.Join(out, "ai-inventory.json"), filepath.Join(out, "ai-inventory.csv")
	if err := os.WriteFile(js, raw, 0o600); err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "cannot write %s: %v", js, err))
	}
	if err := os.WriteFile(csv, inv.Inventory.CSV(), 0o600); err != nil {
		return e.fail(name, errs.New(errs.EUSAGE, "cannot write %s: %v", csv, err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"json": js, "csv": csv, "items": len(inv.Inventory.Items), "sha256": inv.SHA256, "key": inv.Key})
	}
	fmt.Fprintf(e.out, "inventory: %d items in %s and %s, sha256 %s, signed by %s, verified\n", len(inv.Inventory.Items), js, csv, inv.SHA256, inv.Key)
	return 0
}
