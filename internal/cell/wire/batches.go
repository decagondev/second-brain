package wire

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"example.com/second-brain/internal/core/ask"
	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/manifest"
	"example.com/second-brain/internal/core/ports"
)

// batchStore keeps the batches scheduled steps submitted, one file each under
// state/batches (sb-runtime(7), Batching; SB-5701): ids, provider, caller and step, never a
// prompt, so a restart between submit and collect loses nothing.
type batchStore struct{ dir string }

var _ ports.BatchStore = batchStore{}

var batchKeyRE = regexp.MustCompile(`^[0-9a-f]{32}$`)

func (b batchStore) path(key string) (string, error) {
	if !batchKeyRE.MatchString(key) {
		return "", errors.New("batches: not a batch key")
	}
	return filepath.Join(b.dir, key+".json"), nil
}

func (b batchStore) GetBatch(_ context.Context, key string) (ports.PendingBatch, bool, error) {
	p, err := b.path(key)
	if err != nil {
		return ports.PendingBatch{}, false, err
	}
	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return ports.PendingBatch{}, false, nil
	}
	if err != nil {
		return ports.PendingBatch{}, false, err
	}
	var pb ports.PendingBatch
	if err := json.Unmarshal(raw, &pb); err != nil {
		return ports.PendingBatch{}, false, err
	}
	return pb, true, nil
}

func (b batchStore) PutBatch(_ context.Context, pb ports.PendingBatch) error {
	p, err := b.path(pb.Key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(b.dir, 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(pb)
	if err != nil {
		return err
	}
	if err := os.WriteFile(p+".tmp", raw, 0o600); err != nil {
		return err
	}
	return os.Rename(p+".tmp", p)
}

func (b batchStore) DeleteBatch(_ context.Context, key string) error {
	p, err := b.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (b batchStore) ListBatches(ctx context.Context) ([]ports.PendingBatch, error) {
	ents, err := os.ReadDir(b.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []ports.PendingBatch
	for _, e := range ents {
		key, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !batchKeyRE.MatchString(key) {
			continue
		}
		pb, found, err := b.GetBatch(ctx, key)
		if err != nil {
			return nil, err
		}
		if found {
			out = append(out, pb)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (c *Cell) batches() batchStore { return batchStore{filepath.Join(c.Config.State, "batches")} }

// sweepBatches cancels the main tier's batches past their morning deadline (SB-5701).
func (c *Cell) sweepBatches(ctx context.Context) error {
	if f := c.fallback.Load(); f != nil {
		return f.SweepBatches(ctx)
	}
	return nil
}

// laterBatches gives a gateway the cell's batch store and zone (SB-5701), and its
// provider's queue (SB-5703): one per provider and base, shared by its tiers and kept
// across manifest applies.
func (c *Cell) laterBatches(m ports.Model, zone *time.Location, base string) ports.Model {
	g, ok := m.(gateway.Gateway)
	if !ok {
		return m
	}
	g.Batches, g.Zone = c.batches(), zone
	q, _ := c.busy.LoadOrStore(g.Provider+"|"+base, &gateway.Busy{})
	g.Busy = q.(*gateway.Busy)
	return g
}

// deepConfig is cfg with the deep tier's id (when it names one) and price (SB-5704).
func deepConfig(cfg Config) Config {
	mc := cfg
	if d := cfg.Model.Deep; d.ID != "" {
		mc.Model.ID = d.ID
	}
	mc.Model.Price.InPerM, mc.Model.Price.OutPerM = cfg.Model.Deep.Price.InPerM, cfg.Model.Deep.Price.OutPerM
	return mc
}

// withEffort is m asking every call for effort, when m is a gateway.
func withEffort(m ports.Model, effort string) ports.Model {
	if g, ok := m.(gateway.Gateway); ok {
		g.Effort = effort
		return g
	}
	return m
}

// tiers are the small and deep tiers for m, each nil when the cell has none.
func (c *Cell) tiers(m manifest.Tenant) (small, deep ports.Model, err error) {
	if small, err = c.small(m); err != nil {
		return nil, nil, err
	}
	deep, err = c.deep(m)
	return small, deep, err
}

// tracesOf are one person's traces from a time on (SB-5802).
func (c *Cell) tracesOf(ctx context.Context, user string, from time.Time) ([]ask.Trace, error) {
	ts, err := c.Traces.Between(ctx, c.Config.Tenant, from, time.Now().Add(time.Minute))
	if err != nil {
		return nil, err
	}
	var out []ask.Trace
	for _, t := range ts {
		if t.User == user {
			out = append(out, t)
		}
	}
	return out, nil
}
