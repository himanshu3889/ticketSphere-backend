package baseMetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"boundedKey"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"boundedKey"},
	)

	// CacheOperations unified counter.
	// Labels: boundedKey
	CacheOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations categorized by their final state",
		},
		[]string{"boundedKey", "state"},
	)

	CacheLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_latency_seconds",
			Help:    "Cache operation latency",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"boundedKey", "result"}, // result = "hit", "miss"
	)

	// BloomOperations tracks Bloom filter checks.
	BloomOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bloom_operations_total",
			Help: "Total number of Bloom filter operations",
		},
		[]string{"filterKey", "state"},
	)
)
