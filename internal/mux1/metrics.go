package mux1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

type RequestMonitor struct {
	http.Handler
}

var (
	gauge = metrics.NewGauge("active_requests", nil)
	hist  = metrics.NewHistogram("request_latency_ms")
)

func (rm RequestMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	counter := metrics.GetOrCreateCounter(fmt.Sprintf("http_request{method=\"%s\",path=\"%s\"}", r.Method, r.URL.EscapedPath()))
	counter.Inc()

	gauge.Inc()
	defer gauge.Dec()
	start := time.Now()
	defer hist.UpdateDuration(start)
	rm.Handler.ServeHTTP(w, r)
}

type SMetric interface {
	fmt.Stringer
	Metric
}

type Metrics struct {
	mu      sync.RWMutex
	metrics map[string]SMetric
}

var DefaultMetrics = &Metrics{
	metrics: make(map[string]SMetric),
}

func Count(name string) *Counter {

	DefaultMetrics.mu.RLock()
	counter, ok := DefaultMetrics.metrics[name]
	DefaultMetrics.mu.RUnlock()

	if !ok {
		DefaultMetrics.mu.Lock()
		counter, ok = DefaultMetrics.metrics[name]
		defer DefaultMetrics.mu.Unlock()
		if !ok {
			counter = &Counter{
				name:  name,
				count: atomic.Int64{},
			}
			DefaultMetrics.metrics[name] = counter
		}
	}
	if c, ok := counter.(*Counter); ok {
		return c
	}

	return nil

}

func GaugeM(name string) *Gauge {
	DefaultMetrics.mu.RLock()
	gauge, ok := DefaultMetrics.metrics[name]
	DefaultMetrics.mu.RUnlock()
	if !ok {
		DefaultMetrics.mu.Lock()
		gauge, ok = DefaultMetrics.metrics[name]
		defer DefaultMetrics.mu.Unlock()
		if !ok {
			gauge = &Gauge{
				name:  name,
				count: atomic.Int64{},
			}
			DefaultMetrics.metrics[name] = gauge
		}
	}
	if g, ok := gauge.(*Gauge); ok {
		return g
	}
	return nil

}

func HistogramM(name string, min int64, max int64, numBuckets int64) *Histogram {
	DefaultMetrics.mu.RLock()
	histogram, ok := DefaultMetrics.metrics[name]
	DefaultMetrics.mu.RUnlock()
	if !ok {
		DefaultMetrics.mu.Lock()
		histogram, ok = DefaultMetrics.metrics[name]
		defer DefaultMetrics.mu.Unlock()
		if !ok {

			buckets := make([]atomic.Int64, numBuckets)
			for i := range buckets {
				buckets[i] = atomic.Int64{}
			}

			histogram = &Histogram{
				name:       name,
				min:        min,
				max:        max,
				numBuckets: numBuckets,
				buckets:    buckets,
			}
			DefaultMetrics.metrics[name] = histogram
		}
	}
	if h, ok := histogram.(*Histogram); ok {
		return h
	}
	return nil
}

func (m *Metrics) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, v := range m.metrics {
		fmt.Fprintf(writer, "%s\n", v)
	}
	fmt.Fprintf(writer, "\n")
	writer.Header().Set("Content-Type", "text/plain")
}

type Metric interface {
	Name() string
	Value() string
	Timestamp() int64
}

type Counter struct {
	name  string
	count atomic.Int64
}

func (c *Counter) Name() string {
	return c.name
}

func (c *Counter) Value() string {
	return strconv.FormatInt(c.count.Load(), 10)
}

func (c *Counter) Timestamp() int64 {
	return time.Now().Unix()
}
func (c *Counter) Increment() {
	c.count.Add(1)
}

func (c *Counter) String() string {
	sb := strings.Builder{}
	sb.WriteString(c.Name())
	sb.WriteString(" ")
	sb.WriteString(c.Value())
	sb.WriteString(" ")
	sb.WriteString(strconv.FormatInt(c.Timestamp(), 10))
	sb.WriteString("\n")
	return sb.String()

}

type Gauge struct {
	name  string
	count atomic.Int64
}

func (g *Gauge) String() string {

	sb := strings.Builder{}
	sb.WriteString(g.Name())
	sb.WriteString(" ")
	sb.WriteString(g.Value())
	sb.WriteString(" ")
	sb.WriteString(strconv.FormatInt(g.Timestamp(), 10))
	sb.WriteString("\n")
	return sb.String()

}

func (g *Gauge) Name() string {
	return g.name
}

func (g *Gauge) Value() string {
	return strconv.FormatInt(g.count.Load(), 10)
}

func (g *Gauge) Timestamp() int64 {
	return time.Now().Unix()
}

func (g *Gauge) Increment() {
	g.count.Add(1)
}
func (g *Gauge) Decrement() {
	g.count.Add(-1)
}

type Histogram struct {
	name       string
	min        int64
	max        int64
	numBuckets int64
	buckets    []atomic.Int64
}

func (h *Histogram) String() string {

	var sb strings.Builder
	name := h.Name()
	ts := h.Timestamp()
	step := (h.max - h.min + 1) / h.numBuckets

	var cumulative int64

	for i := range h.buckets {
		limit := h.min + int64(i)*step
		cumulative += h.buckets[i].Load()
		fmt.Fprintf(&sb, "%s_bucket{le=\"%d\"} %d %d\n", name, limit, cumulative, ts)
	}

	fmt.Fprintf(&sb, "%s_bucket{le=\"+Inf\"} %d %d\n", name, cumulative, ts)
	fmt.Fprintf(&sb, "%s_count %d %d\n", name, cumulative, ts)

	return sb.String()
}

func (h *Histogram) Name() string {
	return h.name
}
func (h *Histogram) Value() string {
	return ""
}

func (h *Histogram) Timestamp() int64 {
	return time.Now().Unix()
}

func (h *Histogram) Record(interval int64) {
	step := (h.max - h.min + 1) / h.numBuckets
	i := (interval - h.min) / step
	h.buckets[min(i, h.numBuckets-1)].Add(1)
}
