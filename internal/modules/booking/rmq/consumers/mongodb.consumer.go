package consumers

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	bookingRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq/consumers/handlers"
)

// runs the worker pool consuming booking.mongodb.queue
func StartMongoDBConsumer(ctx context.Context, conn *amqp091.Connection, logger *logrus.Logger, cacheStore *bookingCacheStore.CacheStore, workerCount int, tracer trace.Tracer) {
	// TODO: make it configurable in yml file
	cfg := baseRabbitMQ.ConsumerConfig{
		ExchangeName:  string(bookingRMQ.ExchangeBooking),
		RoutingKey:    string(bookingRMQ.BindingKeyBookingAll),
		QueueName:     string(bookingRMQ.QueueBookingMongoDB),
		DLXExchange:   string(bookingRMQ.ExchangeDLX),
		DLXRoutingKey: string(bookingRMQ.BindingKeyBookingAll),
		PrefetchCount: 1, // each worker holds only 1 unacked message
	}

	handler := func(ctx context.Context, msg amqp091.Delivery) error {
		switch msg.RoutingKey {
		case string(bookingRMQ.RoutingKeyBookingCreated):
			return handlers.HandleBookingCreatedMongoDB(ctx, msg.Body, cacheStore, logger)
		case string(bookingRMQ.RoutingKeyBookingConfirmed):
			return handlers.HandleBookingConfirmedMongoDB(ctx, msg.Body, cacheStore, logger)
		case string(bookingRMQ.RoutingKeyBookingFailed):
			return handlers.HandleBookingFailedMongoDB(ctx, msg.Body, cacheStore, logger)
		default:
			logger.WithField("routing_key", msg.RoutingKey).Warn("no mongodb handler for booking event, acking")
			return nil
		}
	}

	runConsumerPool(ctx, conn, cfg, string(bookingRMQ.QueueBookingMongoDB), workerCount, handler, logger, tracer)
}
