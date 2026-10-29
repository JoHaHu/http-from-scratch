package mux1

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	mu      sync.RWMutex
	metrics map[string]*Label
}

type Label struct {
	Name   string
	Values map[string]string
	Metric Metric
}

func (l Label) String() string {
	var sb strings.Builder
	sb.WriteString(l.Name)

	if len(l.Values) > 0 {
		keys := make([]string, 0, len(l.Values))
		for k := range l.Values {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sb.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			fmt.Fprintf(&sb, `%s="%s"`, k, l.Values[k])
		}
		sb.WriteByte('}')
	}

	sb.WriteByte(' ')
	sb.WriteString(l.Metric.Value())
	sb.WriteByte(' ')
	sb.WriteString(strconv.FormatInt(l.Metric.Timestamp(), 10))

	return sb.String()
}

var DefaultMetrics = &Metrics{
	metrics: make(map[string]*Label),
}

func Count(name string, values map[string]string) *Counter {

	DefaultMetrics.mu.RLock()
	counter, ok := DefaultMetrics.metrics[name]
	DefaultMetrics.mu.RUnlock()

	if !ok {
		DefaultMetrics.mu.Lock()
		counter, ok = DefaultMetrics.metrics[name]
		defer DefaultMetrics.mu.Unlock()
		if !ok {
			counter = &Label{
				Name:   name,
				Values: values,
				Metric: &Counter{
					count: atomic.Int64{},
				},
			}
			DefaultMetrics.metrics[name] = counter
		}
	}
	if c, ok := counter.Metric.(*Counter); ok {
		return c
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
	Value() string
	Timestamp() int64
}

type Counter struct {
	count atomic.Int64
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
