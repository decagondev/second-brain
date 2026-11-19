package obs

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Metrics is a small Prometheus text-format registry (gauges and counters with labels).
// It holds ids and numbers only, never content.
type Metrics struct {
	mu     sync.Mutex
	values map[string]float64 // "name{labels}" -> value
	help   map[string]string
	kind   map[string]string
}

// NewMetrics returns an empty registry.
func NewMetrics() *Metrics {
	return &Metrics{values: map[string]float64{}, help: map[string]string{}, kind: map[string]string{}}
}

func series(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	ks := make([]string, 0, len(labels))
	for k := range labels {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	parts := make([]string, len(ks))
	for i, k := range ks {
		v := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(labels[k])
		parts[i] = fmt.Sprintf(`%s="%s"`, k, v)
	}
	return name + "{" + strings.Join(parts, ",") + "}"
}

// Set sets a gauge.
func (m *Metrics) Set(name, help string, labels map[string]string, v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.help[name], m.kind[name] = help, "gauge"
	m.values[series(name, labels)] = v
}

// Add adds to a counter.
func (m *Metrics) Add(name, help string, labels map[string]string, v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.help[name], m.kind[name] = help, "counter"
	m.values[series(name, labels)] += v
}

// Get returns one series' value (tests and alerts).
func (m *Metrics) Get(name string, labels map[string]string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.values[series(name, labels)]
	return v, ok
}

// Write renders the registry in Prometheus text format, sorted.
func (m *Metrics) Write(w io.Writer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.help))
	for n := range m.help {
		names = append(names, n)
	}
	sort.Strings(names)
	keys := make([]string, 0, len(m.values))
	for k := range m.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, n := range names {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", n, m.help[n], n, m.kind[n]); err != nil {
			return err
		}
		for _, k := range keys {
			if k == n || strings.HasPrefix(k, n+"{") {
				if _, err := fmt.Fprintf(w, "%s %g\n", k, m.values[k]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Handler serves the registry at /metrics.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_ = m.Write(w)
	})
}

// Freshness turns sync outcomes into the sb-ingest(7) metrics. Call Update with the time
// before each scrape (or on a ticker) to refresh the age gauges.
type Freshness struct {
	M    *Metrics
	mu   sync.Mutex
	last map[[2]string]float64 // (tenant, connector) -> unix seconds of last success
}

const (
	ageMetric      = "sb_ingest_last_success_age_seconds"
	failuresMetric = "sb_ingest_sync_failures_total"
)

func (f *Freshness) SyncSucceeded(tenant, connector string, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.last == nil {
		f.last = map[[2]string]float64{}
	}
	f.last[[2]string{tenant, connector}] = float64(at.Unix())
}

func (f *Freshness) SyncFailed(tenant, connector string) {
	f.M.Add(failuresMetric, "Failed ingest syncs.", map[string]string{"tenant": tenant, "connector": connector}, 1)
}

// Update sets each connector's age gauge as of now (unix seconds).
func (f *Freshness) Update(now int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, at := range f.last {
		f.M.Set(ageMetric, "Seconds since the last successful ingest sync.", map[string]string{"tenant": k[0], "connector": k[1]}, float64(now)-at)
	}
}
