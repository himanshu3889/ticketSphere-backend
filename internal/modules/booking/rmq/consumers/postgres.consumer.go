package consumers

import (
	"context"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	bookingRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq"
)

// StartPostgresConsumer runs the worker pool consuming booking.postgres.queue.
func StartPostgresConsumer(ctx context.Context, conn *amqp091.Connection, logger *logrus.Logger, cacheStore *bookingCacheStore.CacheStore, tuning configs.ConsumerTuning, tracer trace.Tracer) {
	workers := tuning.Workers
	if workers <= 0 {
		workers = 2
	}
	prefetch := tuning.Prefetch
	if prefetch <= 0 {
		prefetch = 250
	}
	flushEvery := time.Duration(tuning.FlushIntervalMs) * time.Millisecond

	batcher := newPostgresBatcher(cacheStore.Store, logger, tuning.BatchSize, flushEvery)

	cfg := baseRabbitMQ.ConsumerConfig{
		ExchangeName:  string(bookingRMQ.ExchangeBooking),
		RoutingKey:    string(bookingRMQ.BindingKeyBookingAll),
		QueueName:     string(bookingRMQ.QueueBookingPostgres),
		DLXExchange:   string(bookingRMQ.ExchangeDLX),
		DLXRoutingKey: string(bookingRMQ.BindingKeyBookingAll),
		PrefetchCount: prefetch,
		// Quorum disabled: Raft fsync
		// QueueType:     "quorum",
		// DeliveryLimit: 3,
		ManualAck: true,
		OnStop:    batcher.FlushAndStop,
	}

	handler := func(ctx context.Context, msg amqp091.Delivery) error {
		switch msg.RoutingKey {
		case string(bookingRMQ.RoutingKeyBookingCreated):
			return batcher.AddEvent(batchGroupCreated, msg)
		case string(bookingRMQ.RoutingKeyBookingConfirmed):
			return batcher.AddEvent(batchGroupConfirmed, msg)
		case string(bookingRMQ.RoutingKeyBookingFailed):
			return batcher.AddEvent(batchGroupFailed, msg)
		default:
			// ManualAck: never leave a delivery unacked — preserve the old
			// "ack and move on" behaviour for unknown event types.
			logger.WithField("routing_key", msg.RoutingKey).Warn("no postgres batch group for booking event, acking")
			return msg.Ack(false)
		}
	}

	runConsumerPool(ctx, conn, cfg, string(bookingRMQ.QueueBookingPostgres), workers, handler, logger, tracer)
}
