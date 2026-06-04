package baseMetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	KafkaProducerSuccessMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_success_messages",
			Help: "Total kafka producer succeed messages",
		}, []string{"topic"},
	)
	KafkaProducerFailedMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_failed_messages",
			Help: "Total kafka producer failed messages",
		}, []string{"topic"},
	)
	KafkaConsumerSuccessMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_success_messages",
			Help: "Total kafka consumer succeed messages",
		}, []string{"topic"},
	)
	KafkaConsumerFailedMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_failed_messages",
			Help: "Total kafka consumer failed messages",
		}, []string{"topic"},
	)
)
