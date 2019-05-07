package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	Namespace = "zdns"
	Subsystem = "vanguard"
)

var TimeBuckets = prometheus.ExponentialBuckets(0.00025, 2, 16)
var (
	RequestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "request_count_total",
		Help:      "Counter of DNS requests made per view.",
	}, []string{"module", "view"})

	RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "request_duration_seconds",
		Buckets:   TimeBuckets,
		Help:      "Histogram of the time (in seconds) each request took.",
	}, []string{"module", "view"})

	ResponseCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "response_count_total",
		Help:      "Counter of DNS responses made per view.",
	}, []string{"module", "view"})

	UpdateCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "update_count_total",
		Help:      "Counter of DNS update made per view.",
	}, []string{"module", "view"})

	QPS = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "qps",
		Help:      "requests per second, view.",
	}, []string{"module", "view"})

	CacheSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "cache_size",
		Help:      "The number of elements in the cache per view.",
	}, []string{"module", "view"})

	CacheHits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "cache_hits_total",
		Help:      "The count of cache hits per view.",
	}, []string{"module", "view"})
)
