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
		Help:      "Counter of DNS requests made all views.",
	}, []string{"module"})

	RequestCountByView = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "request_count_by_view",
		Help:      "Counter of DNS requests made per view.",
	}, []string{"module", "view"})

	ResponseCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "response_count_total",
		Help:      "Counter of DNS responses made all views.",
	}, []string{"module"})

	ResponseCountByView = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "response_count_by_view",
		Help:      "Counter of DNS responses made per view.",
	}, []string{"module", "view"})

	UpdateCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "update_count_total",
		Help:      "Counter of DNS update made all views.",
	}, []string{"module"})

	UpdateCountByView = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "update_count_by_view",
		Help:      "Counter of DNS update made per view.",
	}, []string{"module", "view"})

	QPS = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "qps",
		Help:      "requests per second made all views.",
	}, []string{"module"})

	QPSByView = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "qps_by_view",
		Help:      "requests per second made per view.",
	}, []string{"module", "view"})

	CacheSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "cache_size_total",
		Help:      "The number of elements in the cache made all views.",
	}, []string{"module"})

	CacheSizeByView = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "cache_size_by_view",
		Help:      "The number of elements in the cache made per view.",
	}, []string{"module", "view"})

	CacheHits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "cache_hits_total",
		Help:      "The count of cache hits all views.",
	}, []string{"module"})

	CacheHitsByView = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "cache_hits_by_view",
		Help:      "The count of cache hits per view.",
	}, []string{"module", "view"})
)
