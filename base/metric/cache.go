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

	CacheLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_latency_seconds",
			Help:    "Cache operation latency",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"boundedKey", "result"}, // result = "hit", "miss"
	)
)
