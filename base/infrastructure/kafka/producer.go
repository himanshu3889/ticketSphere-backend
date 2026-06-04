package baseKafka

// https://medium.com/@harshithgowdakt/kafka-with-confluent-kafka-go-a-go-developers-playbook-30f4993f5248
import (
	"context"
	"fmt"
	"time"

	baseMetrics "github.com/himanshu3889/ticketSphere-backend/base/metric"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"

	"github.com/bwmarrin/snowflake"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Producer sends any struct to any topic
type KafkaProducer struct {
	writer        *kafka.Writer
	snowflakeNode *snowflake.Node
}

// TODO: implement the metric for that also

// New kafka producer
func NewProducer(brokers []string, logger *logrus.Logger) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.Hash{}, // Keep this - ensures same key → same partition

			// SPEED OPTIMIZATION 1: Don't wait for all replicas (local dev only!)
			// Use RequireOne for speed, RequireAll for safety
			RequiredAcks: kafka.RequireOne, // Change to RequireAll if you need durability

			// SPEED OPTIMIZATION 2: Batch settings
			BatchSize:    100,                   // Accumulate 100 messages before sending
			BatchTimeout: 10 * time.Millisecond, // Or flush after 10ms

			// SPEED OPTIMIZATION 3: Async mode - Fire and forget (FASTEST)
			// Returns immediately, batches in background
			Async: true,

			// SPEED OPTIMIZATION 4: Compression (essential for batches)
			Compression: kafka.Snappy, // Fast compression, good ratio
			// Or use kafka.Lz4 for higher compression, more CPU

			// Retry failed sends in background
			MaxAttempts: 3,

			// Error handling for async mode (CRITICAL)
			// Async failures go here instead of returning error
			ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
				logger.Errorf("[KAFKA-PRODUCER] "+msg, args...)
			}),
		},
	}
}

// Write bulk messages to kafka
func (kafkaProducer *KafkaProducer) WriteMessages(ctx context.Context, topic string, messages []kafka.Message) (sendError error) {
	defer func() {
		kafkaProducer.MessagesMetric(topic, len(messages), sendError != nil)
	}()
	return kafkaProducer.writer.WriteMessages(ctx, messages...)
}

// Send puts any data on any topic.
// Key is used for partitioning (same key = same partition = ordering)
// Send produces a message. It accepts optional headers to support context propagation.
func (kafkaProducer *KafkaProducer) Send(ctx context.Context, topic, key string, data []byte, userID snowflake.ID, extraHeaders ...kafka.Header) (sendError error) {

	defer func() {
		kafkaProducer.MessagesMetric(topic, 1, sendError != nil)
	}()

	// Default headers we always want
	headers := []kafka.Header{
		{Key: "user_id", Value: []byte(userID.String())},
		{Key: "publish_time", Value: []byte(fmt.Sprintf("%d", time.Now().UnixMilli()))}, // When *this* specific hop happened
	}

	// Helper: Check if specific keys were provided in extraHeaders
	// NOTE: immutable, You must copy-paste them from the incoming message to the outgoing message every time you republish
	hasTraceID := false
	hasIngestTime := false

	for _, h := range extraHeaders {
		if string(h.Key) == "trace_id" {
			hasTraceID = true
		}
		if string(h.Key) == "ingest_time" {
			hasIngestTime = true
		}
		headers = append(headers, h)
	}

	// If no Trace ID passed (New Message), generate one
	if !hasTraceID {
		headers = append(headers, kafka.Header{
			Key: "trace_id", Value: []byte(utils.GenerateSnowflakeID(kafkaProducer.snowflakeNode).String()),
		})
	}

	// If no Ingest Time passed (New Message), use Now
	if !hasIngestTime {
		headers = append(headers, kafka.Header{
			Key: "ingest_time", Value: []byte(fmt.Sprintf("%d", time.Now().UnixMilli())),
		})
	}

	return kafkaProducer.writer.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   data,
		Headers: headers,
	})
}

// Close kafka producer
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

// Success messages metric
func (p *KafkaProducer) MessagesMetric(topic string, cnt int, isFailed bool) {
	if cnt <= 0 {
		return
	}
	if isFailed {
		p.FailureMessagesMetric(topic, cnt)
	} else {
		p.SuccessMessagesMetric(topic, cnt)
	}
}

// Failure messages metric
func (p *KafkaProducer) FailureMessagesMetric(topic string, cnt int) {
	if cnt <= 0 {
		return
	}
	baseMetrics.KafkaProducerFailedMessages.WithLabelValues(topic).Inc()
}

// Success messages metric
func (p *KafkaProducer) SuccessMessagesMetric(topic string, cnt int) {
	if cnt <= 0 {
		return
	}
	baseMetrics.KafkaProducerSuccessMessages.WithLabelValues(topic).Inc()
}
