package metrics

import "github.com/prometheus/client_golang/prometheus"

// MetricType represents the type of Prometheus metric (Counter or Gauge)
type MetricType int

const (
	Counter MetricType = iota
	Gauge
)

// MetricConfig holds the configuration for a single metric
type MetricConfig struct {
	Name   string
	Help   string
	Labels []string
	Type   MetricType
}

// MetricBuilder helps in building and registering Prometheus metrics
type MetricBuilder struct {
	metrics map[string]interface{}
}

// NewMetricBuilder creates a new MetricBuilder instance
func NewMetricBuilder() *MetricBuilder {
	return &MetricBuilder{
		metrics: make(map[string]interface{}),
	}
}

// AddMetric adds a metric to the builder
func (b *MetricBuilder) AddMetric(config MetricConfig) *MetricBuilder {
	switch config.Type {
	case Counter:
		b.metrics[config.Name] = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: config.Name,
			Help: config.Help,
		}, config.Labels)
	case Gauge:
		b.metrics[config.Name] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: config.Name,
			Help: config.Help,
		}, config.Labels)
	}
	return b
}

// RegisterMetrics registers all metrics with Prometheus
func (b *MetricBuilder) RegisterMetrics() {
	for _, metric := range b.metrics {
		switch m := metric.(type) {
		case *prometheus.CounterVec:
			prometheus.MustRegister(m)
		case *prometheus.GaugeVec:
			prometheus.MustRegister(m)
		}
	}
}

// GetCounterVec returns a specific CounterVec metric by name
func (b *MetricBuilder) GetCounterVec(name string) *prometheus.CounterVec {
	if metric, ok := b.metrics[name].(*prometheus.CounterVec); ok {
		return metric
	}
	return nil
}

// GetGaugeVec returns a specific GaugeVec metric by name
func (b *MetricBuilder) GetGaugeVec(name string) *prometheus.GaugeVec {
	if metric, ok := b.metrics[name].(*prometheus.GaugeVec); ok {
		return metric
	}
	return nil
}

func InitializeMetrics() *MetricBuilder {
	builder := NewMetricBuilder()

	builder.AddMetric(MetricConfig{Name: "schednex_k8sgpt_object_backoff",
		Help:   "The number of times schednex has attempted to find K8sGPT CR",
		Labels: []string{"k8sgpt", "custom_resource"}, Type: Counter})
	builder.AddMetric(MetricConfig{Name: "schednex_k8sgpt_interconnect_backoff",
		Help:   "The number of times schednex has attempted to connect to K8sGPT",
		Labels: []string{"k8sgpt", "interconnect"}, Type: Counter})
	builder.AddMetric(MetricConfig{Name: "schednex_pods_scheduled",
		Help:   "The number of times schednex has scheduled a pod",
		Labels: []string{"schednex"}, Type: Counter})
	builder.RegisterMetrics()

	return builder
}
