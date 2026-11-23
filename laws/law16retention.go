package laws

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/logs/localfs"
	tracemem "example.com/second-brain/internal/adapters/traces/memory"
	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/audit"
	"example.com/second-brain/internal/core/retention"
)

// LAW-16 for the retention plan (SB-1905): what the plan says the sweep will delete is what
// the sweep deletes, store by store, held records counted apart, and the plan deletes
// nothing. LAW-04 and LAW-07 for the subject export (SB-1903): an export is audited before
// it reads anything, holds exactly what erase then removes, leaves its hash and never its
// text in the audit log, and after the erase is empty.
//
// Both run the real job over the real in-memory derived stores and the real log files, in
// random worlds of six record kinds, spaces with shorter retention, and legal holds.

// RetentionStores is a hook for the self-tests: it gets the real stores and returns the
// ones the job runs over.
type RetentionStores func([]retention.Store) []retention.Store

// RealRetentionStores runs the real stores as they are.
var RealRetentionStores RetentionStores = func(s []retention.Store) []retention.Store { return s }

type recordingAudit struct{ rows []audit.Event }

func (a *recordingAudit) Write(_ context.Context, e audit.Event) (audit.Row, error) {
	a.rows = append(a.rows, e)
	return audit.Row{ID: fmt.Sprintf("aud_%012d", len(a.rows)), Kind: e.Kind}, nil
}

// retentionWorldOf fills the real stores with random records and returns a job over them,
// the users, and a cleanup.
func retentionWorldOf(r *rand.Rand, hook RetentionStores) (retention.Job, *recordingAudit, []string, func(), error) {
	dir, err := os.MkdirTemp("", "law16plan")
	if err != nil {
		return retention.Job{}, nil, nil, nil, err
	}
	ctx := context.Background()
	now := time.Date(2027, 3, 10, 3, 0, 0, 0, time.UTC)
	tr := tracemem.New()
	logs := &localfs.Files{Dir: dir}
	users := []string{"usr_ann", "usr_bob", "usr_cy"}
	spaces := []string{"spc_finance", "spc_people", "company"}
	for i := 0; i < 4+r.Intn(20); i++ {
		u, sp := users[r.Intn(len(users))], spaces[r.Intn(len(spaces))]
		at := now.Add(-time.Duration(r.Intn(800*24)) * time.Hour)
		id := fmt.Sprintf("tr_%d", i)
		marker := fmt.Sprintf("secretquestion%d", i)
		tr.Now = func() time.Time { return at }
		switch r.Intn(5) {
		case 0:
			_ = tr.PutTrace(ctx, ask.Trace{ID: id, Tenant: "acme-logistics", User: u, Space: sp, At: at})
			_ = tr.PutFeedback(ctx, ask.Feedback{Tenant: "acme-logistics", User: u, Trace: id, At: at, Comment: marker})
		case 1:
			_ = tr.AppendQuestion(ctx, "acme-logistics", u, "cv_"+id, marker)
			_ = tr.TagSpace(ctx, "acme-logistics", "cv_"+id, sp)
		case 2:
			_ = tr.PutSaved(ctx, ask.SavedQuestion{ID: "sq_" + id, Tenant: "acme-logistics", User: u, Question: marker, Space: sp, At: at})
		case 3:
			_ = tr.PutShare(ctx, ask.Share{ID: "sh_" + id, Tenant: "acme-logistics", User: u, Question: marker, Space: sp, At: at})
		default:
			_ = tr.PutStewardQuestion(ctx, ask.StewardQuestion{ID: "stq_" + id, Tenant: "acme-logistics", Asker: u, Question: marker, Space: sp, At: at})
		}
		if r.Intn(3) == 0 {
			logs.Now = func() time.Time { return at }
			_, _ = fmt.Fprintf(logs, `{"user":%q,"msg":"ask"}`+"\n", u)
		}
	}
	held := map[string]bool{}
	if r.Intn(2) == 0 {
		held[spaces[r.Intn(2)]] = true
	}
	custodian := ""
	if r.Intn(3) == 0 {
		custodian = users[r.Intn(len(users))]
	}
	spaceDays := map[string]int{}
	if r.Intn(2) == 0 {
		spaceDays["spc_finance"] = 30 + r.Intn(100)
	}
	a := &recordingAudit{}
	j := retention.Job{Tenant: "acme-logistics", Days: 30 + r.Intn(500), Audit: a, Now: func() time.Time { return now },
		Stores:    hook(append(tr.Retention(), logs)),
		SpaceDays: spaceDays,
		Held:      func(space, user string) bool { return held[space] || (custodian != "" && user == custodian) },
	}
	if len(held) == 0 && custodian == "" {
		j.Held = nil
	}
	return j, a, users, func() { _ = os.RemoveAll(dir) }, nil
}

func law16RetentionPlan(r *rand.Rand, hook RetentionStores) error {
	j, _, users, done, err := retentionWorldOf(r, hook)
	if err != nil {
		return err
	}
	defer done()
	ctx := context.Background()
	count := func() (int, error) {
		n := 0
		for _, u := range users {
			for _, s := range j.Stores {
				rs, err := s.Export(ctx, j.Tenant, u)
				if err != nil {
					return 0, err
				}
				n += len(rs)
			}
		}
		return n, nil
	}
	before, err := count()
	if err != nil {
		return err
	}
	plan, err := j.Plan(ctx, 0)
	if err != nil {
		return fmt.Errorf("plan: %v", err)
	}
	if after, err := count(); err != nil || after != before {
		return fmt.Errorf("the plan changed the stores: %d records before, %d after (%v)", before, after, err)
	}
	rep, err := j.Sweep(ctx)
	if err != nil {
		return fmt.Errorf("sweep: %v", err)
	}
	for _, s := range j.Stores {
		if plan.Counts[s.Name()] != rep.Counts[s.Name()] {
			return fmt.Errorf("%s: the plan said %d, the sweep deleted %d (days %d, spaces %v)", s.Name(), plan.Counts[s.Name()], rep.Counts[s.Name()], j.Days, j.SpaceDays)
		}
	}
	if plan.Held != rep.Held {
		return fmt.Errorf("the plan counted %d held, the sweep kept %d", plan.Held, rep.Held)
	}
	if after, _ := count(); before-after != total(rep.Counts) {
		return fmt.Errorf("the sweep reported %d deleted, the stores lost %d", total(rep.Counts), before-after)
	}
	return nil
}

func total(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func law07SubjectExport(r *rand.Rand, hook RetentionStores) error {
	j, a, users, done, err := retentionWorldOf(r, hook)
	if err != nil {
		return err
	}
	defer done()
	j.Held = nil // erase refuses a custodian; the export law is about the copy
	ctx := context.Background()
	u := users[r.Intn(len(users))]
	ex, err := j.Export(ctx, "usr_dpo", u, "access request")
	if err != nil {
		return fmt.Errorf("export: %v", err)
	}
	if len(a.rows) != 2 || a.rows[0].State != "requested" || a.rows[0].Subject != "export:"+u {
		return fmt.Errorf("the export wasn't audited before and after: %+v", a.rows)
	}
	if !strings.Contains(a.rows[1].Reason, "sha256="+ex.Hash()) || ex.SHA256 != ex.Hash() {
		return fmt.Errorf("the outcome row doesn't hold the export's hash: %q", a.rows[1].Reason)
	}
	for _, row := range a.rows {
		if b, _ := json.Marshal(row); strings.Contains(string(b), "secretquestion") {
			return fmt.Errorf("an audit row holds the export's text: %s", b)
		}
	}
	for _, rs := range ex.Records {
		for _, rec := range rs {
			for _, other := range users {
				if other != u && strings.Contains(string(rec), `"`+other+`"`) {
					return fmt.Errorf("the export of %s holds a record of %s: %s", u, other, rec)
				}
			}
		}
	}
	rep, err := j.Erase(ctx, "usr_dpo", u, "erasure request")
	if err != nil {
		return fmt.Errorf("erase: %v", err)
	}
	for _, s := range j.Stores {
		if len(ex.Records[s.Name()]) != rep.Counts[s.Name()] {
			return fmt.Errorf("%s: the export held %d records, erase removed %d", s.Name(), len(ex.Records[s.Name()]), rep.Counts[s.Name()])
		}
	}
	again, err := j.Export(ctx, "usr_dpo", u, "check")
	if err != nil {
		return err
	}
	if again.Count() != 0 {
		return fmt.Errorf("after the erase, the export still holds %d records", again.Count())
	}
	return nil
}
