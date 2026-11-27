package wire

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/second-brain/internal/adapters/model/transcribe"
	"example.com/second-brain/internal/core/ports"
	"example.com/second-brain/internal/core/prompts"
)

// Scans, images and recordings (SB-6101, SB-6102): the vision tier reads a document's
// pictures and the transcribe tier its recordings, each only while the evals job has gated
// it for the release the cell runs (LAW-06), and in the model's region (LAW-33). A part
// neither may read is skipped and counted on the connector's health.

// ErrNotGated is a reader the evals job hasn't gated for this release.
var ErrNotGated = errors.New("wire: the reader isn't gated for this release")

// VisionGatePath and TranscribeGatePath are where the evals job writes the release each
// reader passed.
func VisionGatePath(state string) string { return filepath.Join(state, "evals", "vision-gated") }
func TranscribeGatePath(state string) string {
	return filepath.Join(state, "evals", "transcribe-gated")
}

func (c *Cell) gated(p string) bool {
	b, err := os.ReadFile(p)
	return err == nil && strings.TrimSpace(string(b)) == c.Release
}

// readVision reads one picture with the vision tier.
func (c *Cell) readVision(ctx context.Context, p ports.Part) (string, error) {
	if c.vision == nil || !c.gated(VisionGatePath(c.Config.State)) {
		return "", ErrNotGated
	}
	m, err := c.vision(c.Manifest())
	if err != nil || m == nil {
		return "", ErrNotGated
	}
	pr, err := prompts.Get("vision")
	if err != nil {
		return "", err
	}
	part := p
	resp, err := m.Complete(ctx, ports.ModelRequest{Model: c.Config.Model.Vision.ID, System: pr.Text, MaxTokens: 2000,
		Messages: []ports.Message{{Role: "user", Content: "Read this picture.", Image: &part}}})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// transcriber is the cell's transcribe provider, nil for none.
func (c *Cell) transcriber() ports.Transcriber {
	t := c.Config.Model.Transcribe
	if t.Provider == "" {
		return nil
	}
	hc := c.opt.ConnectorHTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Minute}
	}
	var key func(context.Context) (string, error)
	if !t.Local {
		key = c.Secret("model-transcribe")
	}
	return transcribe.Client{Provider: t.Provider, Base: t.Base, Model: t.ID, Key: key, HTTP: hc}
}

// DefaultMaxMinutes is model.transcribe.max_minutes's default.
const DefaultMaxMinutes = 120

// ErrTooLong is a recording over model.transcribe.max_minutes.
var ErrTooLong = errors.New("wire: the recording is longer than model.transcribe.max_minutes")

// readTranscribe transcribes one recording, if gated and within the cap.
func (c *Cell) readTranscribe(ctx context.Context, p ports.Part) (string, error) {
	tr := c.transcriber()
	if tr == nil || !c.gated(TranscribeGatePath(c.Config.State)) {
		return "", ErrNotGated
	}
	limit := float64(c.Config.Model.Transcribe.MaxMinutes)
	if limit == 0 {
		limit = DefaultMaxMinutes
	}
	if transcribe.Minutes(p) > limit {
		return "", ErrTooLong
	}
	month := time.Now().UTC().Format("2006-01")
	meter := c.Usage.Spend()
	if b := float64(c.Manifest().BudgetUSDMonth); b > 0 {
		if spent, err := meter.Spent(ctx, c.Config.Tenant, month); err != nil || spent >= b {
			return "", ErrNotGated // over the budget, or a budget that can't be read: nothing is sent
		}
	}
	segs, minutes, err := tr.Transcribe(ctx, p)
	if err != nil {
		return "", err
	}
	c.Metrics.Add("sb_transcribe_minutes_total", "Minutes of recordings transcribed (SB-6102).", nil, minutes)
	if usd := c.Config.Model.Transcribe.PerMinuteUSD * minutes; usd > 0 {
		if err := meter.Add(ctx, c.Config.Tenant, month, usd); err != nil {
			return "", err
		}
	}
	if minutes > limit {
		return "", ErrTooLong
	}
	return transcribe.Text(segs), nil
}

// unread counts a part no reader read, on its connector's health (SB-6101, SB-6102).
func (c *Cell) unread(kind, what string) {
	c.Metrics.Add("sb_ingest_parts_unread_total", "Images and recordings no gated reader read (SB-6101, SB-6102).", map[string]string{"connector": kind, "part": what}, 1)
	if err := c.Syncs.Unread(context.Background(), kind, what); err != nil {
		c.Metrics.Add("sb_health_record_failures_total", "Sync records the console reads that couldn't be written (SB-1607).", nil, 1)
	}
}

// digitsProbe is the vision gate's picture: six random digits drawn large, and the digits.
func digitsProbe(r *rand.Rand) (ports.Part, string) {
	digits := make([]byte, 6)
	for i := range digits {
		digits[i] = byte('0' + r.Intn(10))
	}
	return ports.Part{Name: "probe.png", MIME: "image/png", Data: DigitsPNG(string(digits))}, string(digits)
}

// VisionProbes is how many pictures of digits the vision tier must read back.
const VisionProbes = 3

// TranscribeSamples is where an operator puts the recordings the transcribe tier must
// transcribe, each beside a .txt file of the words it must hear (sb-evals(7), Readers).
func TranscribeSamples(state string) string { return filepath.Join(state, "evals", "transcribe") }

// gateReaders writes the vision and transcribe gates for this release (SB-6101, SB-6102):
// vision must read back random digits it is shown, every time; transcribe must hear every
// word of each sample an operator gave, and there must be one. A reader not configured,
// or one that fails, has no gate, so nothing is read with it (LAW-06).
func (c *Cell) gateReaders(ctx context.Context, r *rand.Rand) error {
	if c.Config.Model.Vision.ID != "" {
		ok := c.vision != nil
		if m, err := c.vision(c.Manifest()); ok && err == nil && m != nil {
			pr, _ := prompts.Get("vision")
			for i := 0; i < VisionProbes && ok; i++ {
				part, want := digitsProbe(r)
				resp, err := m.Complete(ctx, ports.ModelRequest{Model: c.Config.Model.Vision.ID, System: pr.Text, MaxTokens: 50,
					Messages: []ports.Message{{Role: "user", Content: "Read this picture.", Image: &part}}})
				ok = err == nil && strings.Contains(strings.ReplaceAll(resp.Text, " ", ""), want)
			}
		} else {
			ok = false
		}
		if err := writeGate(VisionGatePath(c.Config.State), c.Release, ok); err != nil {
			return err
		}
	}
	if tr := c.transcriber(); tr != nil {
		ok := false
		dir := TranscribeSamples(c.Config.State)
		ents, _ := os.ReadDir(dir)
		for _, e := range ents {
			name := e.Name()
			if strings.HasSuffix(name, ".txt") || e.IsDir() {
				continue
			}
			want, err1 := os.ReadFile(filepath.Join(dir, strings.TrimSuffix(name, filepath.Ext(name))+".txt"))
			data, err2 := os.ReadFile(filepath.Join(dir, name))
			if err1 != nil || err2 != nil {
				ok = false
				break
			}
			segs, _, err := tr.Transcribe(ctx, ports.Part{Name: name, Data: data})
			got := strings.ToLower(transcribe.Text(segs))
			ok = err == nil
			for _, w := range strings.Fields(strings.ToLower(string(want))) {
				ok = ok && strings.Contains(got, strings.Trim(w, ".,;:!?"))
			}
			if !ok {
				break
			}
		}
		if err := writeGate(TranscribeGatePath(c.Config.State), c.Release, ok); err != nil {
			return err
		}
	}
	return nil
}
