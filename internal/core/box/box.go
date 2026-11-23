// Package box decides the health of a self-hosted cell's machine from facts the host step
// gathers (sb-selfhost(7), Doctor; SB-1801, SB-1802): its services and timers, its disks,
// its certificate, its off-box directory and its backups. The same verdicts serve
// `install.sh --doctor` on the machine and the console's Box section. A fact that is
// missing is Unknown, and Unknown is never shown as healthy (LAW-03's rule, for health).
package box

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Verdict is one check's result, best first.
type Verdict string

const (
	Pass    Verdict = "pass"
	Warn    Verdict = "warn"
	Fail    Verdict = "fail"
	Unknown Verdict = "unknown"
)

func rank(v Verdict) int {
	switch v {
	case Pass:
		return 0
	case Warn:
		return 1
	case Unknown:
		return 2
	case Fail:
		return 3
	}
	return 3
}

// Thresholds: warn first, fail after.
const (
	DiskWarnPct = 80
	DiskFailPct = 90
	CertWarn    = 21 * 24 * time.Hour
	CertFail    = 7 * 24 * time.Hour
	BackupWarn  = 26 * time.Hour
	BackupFail  = 50 * time.Hour
	FactsMaxAge = 2 * time.Hour
	DrillWarn   = 35 * 24 * time.Hour
	DrillFail   = 62 * 24 * time.Hour
)

// Service is one compose service as `ps` reports it.
type Service struct {
	Name  string `json:"name"`
	State string `json:"state"` // running, exited, restarting, ...
}

// Timer is one systemd timer and the last result of its service.
type Timer struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Result  string `json:"result"` // success, or what systemd says
}

// Disk is one directory's filesystem.
type Disk struct {
	Path    string  `json:"path"`
	UsedPct float64 `json:"used_pct"`
}

// Offsite is the off-box backup directory, or the bucket (SB-2303).
type Offsite struct {
	Path     string `json:"path"`
	Mounted  bool   `json:"mounted"`
	Writable bool   `json:"writable"`
	OnRoot   bool   `json:"on_root"`
	// S3 is an s3:// bucket, and Answer what it said to the key pair (sb backup check):
	// ok, refused or unreachable.
	S3     bool   `json:"s3,omitempty"`
	Answer string `json:"answer,omitempty"`
}

// Facts are what the host step saw. A nil pointer or an empty list is a fact it couldn't
// gather.
type Facts struct {
	At          time.Time  `json:"at"`
	Services    []Service  `json:"services,omitempty"`
	Timers      []Timer    `json:"timers,omitempty"`
	Disks       []Disk     `json:"disks,omitempty"`
	CertExpires *time.Time `json:"cert_expires,omitempty"`
	CertManaged bool       `json:"cert_managed,omitempty"` // ACME: Caddy renews it
	Offsite     *Offsite   `json:"offsite,omitempty"`
	Ready       int        `json:"ready,omitempty"`     // /readyz's status
	Encrypted   *bool      `json:"encrypted,omitempty"` // build/selfhost on an encrypted disk
	LastDump    *time.Time `json:"last_dump,omitempty"`
	LastOffsite *time.Time `json:"last_offsite,omitempty"`
	// The last restore drill (SB-1803) and whether it passed.
	LastDrill *time.Time `json:"last_drill,omitempty"`
	DrillOK   bool       `json:"drill_ok,omitempty"`
	// Held names the release upgrades are held at after a rollback (SB-1805).
	Held string `json:"held,omitempty"`
	// Sealed is whether off-box copies are sealed to a recovery key (SB-1901). Recovery is
	// where that key's private half is: "pending" (still on the box), "kit" (only in the
	// recovery kit's shares), or "none"; SharesOnBox counts share pages not yet deleted
	// from the box (SB-1902).
	Sealed      *bool  `json:"sealed,omitempty"`
	Recovery    string `json:"recovery,omitempty"`
	SharesOnBox int    `json:"shares_on_box,omitempty"`
}

// Parse reads facts and refuses anything it doesn't know, so a typo is never a silent pass.
func Parse(b []byte) (Facts, error) {
	var f Facts
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err := d.Decode(&f); err != nil {
		return Facts{}, fmt.Errorf("box: facts: %w", err)
	}
	if f.At.IsZero() {
		return Facts{}, fmt.Errorf("box: facts: no time")
	}
	return f, nil
}

// Line is one check.
type Line struct {
	Check   string  `json:"check"`
	Verdict Verdict `json:"verdict"`
	Detail  string  `json:"detail"`
	Fix     string  `json:"fix,omitempty"`
}

// Services are the ones a running cell always has.
var Services = []string{"proxy", "api", "ingest", "jobs", "worm", "postgres"}

// Timers are the ones the install enables.
var Timers = []string{"sb-selfhost-update.timer", "sb-selfhost-dbdump.timer", "sb-selfhost-drill.timer"}

// Check decides every check. Facts older than FactsMaxAge make every line Unknown: an old
// green is not a green.
func Check(f Facts, now time.Time) []Line {
	var out []Line
	add := func(check string, v Verdict, detail, fix string) {
		if v == Pass {
			fix = ""
		}
		out = append(out, Line{Check: check, Verdict: v, Detail: detail, Fix: fix})
	}
	stale := f.At.IsZero() || now.Sub(f.At) > FactsMaxAge || f.At.After(now.Add(5*time.Minute))
	checkRuntime(f, add)
	checkStorage(f, now, add)
	checkRecovery(f, now, add)
	checkEdge(f, now, add)
	if stale {
		for i := range out {
			out[i].Verdict, out[i].Fix = Unknown, "check the hourly host step: systemctl status sb-selfhost-update.service"
			out[i].Detail = "facts are older than two hours: " + out[i].Detail
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return rank(out[a].Verdict) > rank(out[b].Verdict) })
	return out
}

// Runbook is the operator's runbook that a fix may point to (SB-2605).
const Runbook = "docs/runbooks/selfhost.md"

// see names a section of the runbook after a fix's command.
func see(section string) string { return " (see " + Runbook + "#" + section + ")" }

type adder func(check string, v Verdict, detail, fix string)

// checkRuntime judges the services and timers.
func checkRuntime(f Facts, add adder) {
	// Services.
	got := map[string]string{}
	for _, s := range f.Services {
		got[s.Name] = s.State
	}
	for _, name := range Services {
		st, ok := got[name]
		switch {
		case len(f.Services) == 0:
			add("service "+name, Unknown, "the host step couldn't list services", "tools/selfhost/compose.sh ps")
		case !ok:
			if name == "postgres" && f.externalDB() {
				continue
			}
			add("service "+name, Fail, "not running", "tools/selfhost/compose.sh up -d "+name)
		case st != "running":
			add("service "+name, Fail, st, "tools/selfhost/compose.sh logs --tail 100 "+name)
		default:
			add("service "+name, Pass, "running", "")
		}
	}
	// Timers.
	tg := map[string]Timer{}
	for _, t := range f.Timers {
		tg[t.Name] = t
	}
	for _, name := range Timers {
		t, ok := tg[name]
		switch {
		case len(f.Timers) == 0:
			add("timer "+name, Unknown, "the host step couldn't read systemd", "systemctl list-timers 'sb-selfhost*'")
		case !ok || !t.Enabled:
			add("timer "+name, Fail, "not enabled", "systemctl enable --now "+name)
		case t.Result != "" && t.Result != "success":
			svc := strings.TrimSuffix(name, ".timer") + ".service"
			add("timer "+name, Fail, "last run: "+t.Result, "journalctl -u "+svc+" -n 50")
		default:
			add("timer "+name, Pass, "enabled", "")
		}
	}
}

// checkStorage judges the disks, the off-box directory and the backups' age.
func checkStorage(f Facts, now time.Time, add adder) {
	// Disks.
	if len(f.Disks) == 0 {
		add("disk", Unknown, "the host step couldn't read disk use", "df -h")
	}
	for _, d := range f.Disks {
		detail := fmt.Sprintf("%.0f%% used", d.UsedPct)
		switch {
		case d.UsedPct >= DiskFailPct:
			add("disk "+d.Path, Fail, detail, "free space on "+d.Path+" (old dumps: build/selfhost/db-dumps)")
		case d.UsedPct >= DiskWarnPct:
			add("disk "+d.Path, Warn, detail, "free space on "+d.Path+" soon")
		default:
			add("disk "+d.Path, Pass, detail, "")
		}
	}
	// Off-box directory.
	switch o := f.Offsite; {
	case o == nil:
		add("offsite", Unknown, "the off-box directory couldn't be checked", "findmnt -T $SB_OFFSITE")
	case o.S3 && o.Answer == "ok":
		add("offsite", Pass, o.Path+" answers with the key pair", "")
	case o.S3 && o.Answer == "refused":
		add("offsite", Fail, o.Path+" refuses the key pair, or has no such bucket", "fix the bucket or SB_OFFSITE_KEY_FILE, then install.sh CONFIG")
	case o.S3:
		add("offsite", Unknown, o.Path+" couldn't be reached", "sb backup check --to "+o.Path+" --keys build/selfhost/vault/jobs/vault/tenants/TENANT/offsite")
	case !o.Mounted || o.OnRoot:
		add("offsite", Fail, o.Path+" is on this machine's root disk, not another machine", "mount the other machine's directory at "+o.Path+see("backup"))
	case !o.Writable:
		add("offsite", Fail, o.Path+" isn't writable by uid 65532", "chown 65532:65532 "+o.Path)
	default:
		add("offsite", Pass, o.Path+" mounted and writable", "")
	}
	// Backups.
	age := func(check string, t *time.Time, fix string) {
		if t == nil {
			add(check, Unknown, "none found", fix)
			return
		}
		a := now.Sub(*t)
		detail := fmt.Sprintf("newest %s (%.0f hours ago)", t.UTC().Format(time.RFC3339), a.Hours())
		switch {
		case a > BackupFail:
			add(check, Fail, detail, fix)
		case a > BackupWarn:
			add(check, Warn, detail, fix)
		default:
			add(check, Pass, detail, "")
		}
	}
	age("database dump", f.LastDump, "systemctl start sb-selfhost-dbdump.service")
	age("off-box copy", f.LastOffsite, "systemctl start sb-selfhost-dbdump.service; check the offsite line")
}

// checkRecovery judges the restore drill and whether upgrades are held.
func checkRecovery(f Facts, now time.Time, add adder) {
	// The restore drill: a dump nobody has restored is a hope, not a backup.
	switch {
	case f.LastDrill == nil:
		add("restore drill", Unknown, "no drill has run", "tools/selfhost/restore.sh --drill"+see("restore"))
	case !f.DrillOK:
		add("restore drill", Fail, "the last drill failed ("+f.LastDrill.UTC().Format("2006-01-02")+")", "tools/selfhost/restore.sh --drill"+see("restore"))
	case now.Sub(*f.LastDrill) > DrillFail:
		add("restore drill", Fail, "the last drill passed "+f.LastDrill.UTC().Format("2006-01-02"), "tools/selfhost/restore.sh --drill"+see("restore"))
	case now.Sub(*f.LastDrill) > DrillWarn:
		add("restore drill", Warn, "the last drill passed "+f.LastDrill.UTC().Format("2006-01-02"), "tools/selfhost/restore.sh --drill"+see("restore"))
	default:
		add("restore drill", Pass, "passed "+f.LastDrill.UTC().Format("2006-01-02"), "")
	}
	checkKit(f, add)
	// Upgrades held after a rollback.
	if f.Held != "" {
		add("upgrades", Warn, "held at "+f.Held+" after a rollback", "tools/selfhost/update.sh --resume"+see("upgrades"))
	} else {
		add("upgrades", Pass, "not held", "")
	}
}

// checkKit judges the sealing of off-box copies and the recovery kit (SB-1901, SB-1902).
func checkKit(f Facts, add adder) {
	switch {
	case f.Sealed == nil:
		add("off-box sealing", Unknown, "couldn't tell whether off-box copies are sealed", "ls build/selfhost/recovery.pub")
	case !*f.Sealed:
		add("off-box sealing", Warn, "off-box copies are not sealed (SB_SEAL=off): whoever reads SB_OFFSITE reads them", "set SB_SEAL=on and run install.sh again")
	default:
		add("off-box sealing", Pass, "off-box copies are sealed to the recovery key", "")
	}
	switch {
	case f.Sealed != nil && !*f.Sealed:
		return
	case f.Recovery == "":
		add("recovery kit", Unknown, "couldn't tell where the recovery key is", "ls build/selfhost/kit")
	case f.Recovery == "pending":
		add("recovery kit", Warn, "the recovery key's private half is still on this box, so the box's loss loses the off-box copies' key and a thief gets both", "tools/selfhost/install.sh --recovery-kit"+see("backup"))
	case f.Recovery == "none":
		add("recovery kit", Fail, "no recovery kit and no recovery key: the sealed off-box copies can't be opened", "tools/selfhost/install.sh --recovery-kit --new-key"+see("backup"))
	case f.SharesOnBox > 0:
		add("recovery kit", Warn, fmt.Sprintf("%d share page(s) are still on this box", f.SharesOnBox), "print them for their custodians, then delete build/selfhost/kit/*-share-*.txt")
	default:
		add("recovery kit", Pass, "the recovery key is only in the kit's shares", "")
	}
}

// checkEdge judges the certificate, /readyz and disk encryption.
func checkEdge(f Facts, now time.Time, add adder) {
	// Certificate.
	switch {
	case f.CertManaged:
		add("certificate", Pass, "Caddy's own (acme or internal), renewed by Caddy", "")
	case f.CertExpires == nil:
		add("certificate", Unknown, "the certificate's expiry couldn't be read", "openssl x509 -enddate -noout -in $SB_TLS_CERT")
	default:
		left := f.CertExpires.Sub(now)
		detail := fmt.Sprintf("expires %s (%d days)", f.CertExpires.UTC().Format("2006-01-02"), int(left.Hours()/24))
		fix := "put the renewed certificate at SB_TLS_CERT and run install.sh again"
		switch {
		case left <= CertFail:
			add("certificate", Fail, detail, fix)
		case left <= CertWarn:
			add("certificate", Warn, detail, fix)
		default:
			add("certificate", Pass, detail, "")
		}
	}
	// Ready.
	switch {
	case f.Ready == 0:
		add("readyz", Unknown, "the host step couldn't reach /readyz", "curl -k https://$SB_HOST/readyz")
	case f.Ready < 200 || f.Ready > 299:
		add("readyz", Fail, fmt.Sprintf("/readyz answered %d", f.Ready), "tools/selfhost/compose.sh logs --tail 100 api")
	default:
		add("readyz", Pass, fmt.Sprintf("/readyz answered %d", f.Ready), "")
	}
	// Disk encryption: a warning, never a refusal.
	switch {
	case f.Encrypted == nil:
		add("encrypted disk", Unknown, "couldn't tell whether build/selfhost is on an encrypted disk", "lsblk -o NAME,TYPE,MOUNTPOINT")
	case !*f.Encrypted:
		add("encrypted disk", Warn, "build/selfhost, which holds the keys, isn't on an encrypted disk", "move /opt/second-brain to a LUKS volume")
	default:
		add("encrypted disk", Pass, "build/selfhost is on an encrypted disk", "")
	}
}

// externalDB is a layout whose database is on another machine (SB-1807): the host step
// lists no postgres service and says so with the marker service "postgres-external".
func (f Facts) externalDB() bool {
	for _, s := range f.Services {
		if s.Name == "postgres-external" {
			return true
		}
	}
	return false
}

// Worst is the worst verdict of the lines; no lines at all is Unknown.
func Worst(lines []Line) Verdict {
	if len(lines) == 0 {
		return Unknown
	}
	w := Pass
	for _, l := range lines {
		if rank(l.Verdict) > rank(w) {
			w = l.Verdict
		}
	}
	return w
}

// ExitCode is the doctor's exit code, as sb(1) numbers them: 1 (a check failed) for any
// failing or unknown line, 0 otherwise; warnings are printed, never hidden.
func ExitCode(lines []Line) int {
	switch Worst(lines) {
	case Pass, Warn:
		return 0
	case Fail, Unknown:
		return 1
	}
	return 1
}
