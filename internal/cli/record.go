package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"example.com/second-brain/internal/cell/wire"
	"example.com/second-brain/internal/core/canary"
	"example.com/second-brain/internal/core/record"
	"example.com/second-brain/internal/errs"
)

var recordFlags = map[string]bool{"--config": true, "--vault": true, "--out": true, "--person": true, "--mailbox": true}

const recordUse = "usage: sb record graph --config FILE [--vault DIR] --out FILE [--person EMAIL] [--mailbox ADDR]"

// record runs a fixed script of reads against a real sandbox and writes a recording in the
// recorded fakes' format (SB-6002; sb(1), sb record), every value replaced. The recording
// is scanned before it is written: a stored secret, a secret's shape or an address that
// isn't a placeholder person refuses it.
func (e *env) record(args []string) int {
	const cmd = "sb record"
	args = e.localArgs(args)
	if len(args) == 0 || args[0] != "graph" {
		return e.fail(cmd, errs.New(errs.EUSAGE, recordUse))
	}
	opt := map[string]string{}
	for i := 1; i < len(args); i++ {
		a := args[i]
		if !recordFlags[a] || i+1 >= len(args) {
			return e.fail(cmd, errs.New(errs.EUSAGE, recordUse))
		}
		opt[a] = args[i+1]
		i++
	}
	if opt["--config"] == "" || opt["--out"] == "" {
		return e.fail(cmd, errs.New(errs.EUSAGE, recordUse))
	}
	b, err := os.ReadFile(opt["--config"])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "can't read %s", opt["--config"]))
	}
	cfg, err := wire.Parse(b)
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%s: %v", opt["--config"], err))
	}
	vault := opt["--vault"]
	if vault == "" {
		vault = cfg.Vault
	}
	g, _ := wire.GraphFor(cfg, vault, &http.Client{Timeout: 30 * time.Second})
	ex, err := g.Script(context.Background(), cfg.Microsoft.Drives, opt["--person"], opt["--mailbox"])
	if err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	var steps []record.Exchange
	for _, x := range ex {
		steps = append(steps, record.Exchange{Step: x.Step, Status: x.Status, Body: x.Body})
	}
	rec := record.New("graph", "graph-v1.0", time.Now(), steps, record.NewScrubber(time.Now().UnixNano()))
	out, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return e.fail(cmd, err)
	}
	var s canary.Scanner
	_ = walkFiles(vault, func(_ string, b []byte) { s.Learn(string(b)) })
	if f := s.ScanAll("recording", string(out)); len(f) > 0 {
		return e.fail(cmd, errs.New(errs.ELAW, "the recording holds a secret or a secret's shape (%s); nothing written", f[0].ID))
	}
	if err := record.CheckPeople(out); err != nil {
		return e.fail(cmd, errs.New(errs.ELAW, "%v; nothing written", err))
	}
	if err := os.WriteFile(opt["--out"], append(out, '\n'), 0o600); err != nil {
		return e.fail(cmd, errs.New(errs.EINVAL, "%v", err))
	}
	if e.g.JSON {
		return e.json(map[string]any{"file": opt["--out"], "exchanges": len(rec.Exchanges)})
	}
	fmt.Fprintf(e.out, "record: %d exchanges, every value replaced, in %s\n", len(rec.Exchanges), opt["--out"])
	return 0
}
