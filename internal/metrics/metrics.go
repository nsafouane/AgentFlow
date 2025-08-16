// Package metrics provides observability and metrics collection for AgentFlow
package metrics

// Metrics provides interface for collecting application metrics
type Metrics interface {
	Counter(name string) Counter
	Histogram(name string) Histogram
}

// Counter represents a monotonic counter metric
type Counter interface {
	Inc()
	Add(value float64)
}

// Histogram represents a histogram metric
type Histogram interface {
	Observe(value float64)
}

// NewMetrics creates a new metrics collector
func NewMetrics() Metrics {
	// Metrics implementation will be added
	return &noopMetrics{}
}

type noopMetrics struct{}

func (m *noopMetrics) Counter(name string) Counter {
	return &noopCounter{}
}

func (m *noopMetrics) Histogram(name string) Histogram {
	return &noopHistogram{}
}

type noopCounter struct{}

func (c *noopCounter) Inc()              {}
func (c *noopCounter) Add(value float64) {}

type noopHistogram struct{}

func (h *noopHistogram) Observe(value float64) {}
