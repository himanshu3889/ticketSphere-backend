package bookingConsumers

import (
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// Start all consumers for booking module
func StartConsumers(conn *amqp091.Connection, cachedStore *bookingCacheStore.CacheStore, logger *logrus.Logger, traceProvider trace.TracerProvider) error {
	// Create one tracer for the entire consumer layer
	consumerTracer := traceProvider.Tracer("ticketSphere.booking.consumers")

	// consumers
	workerCount := 2 // TODO: Think about it
	go BookingPostgresConsumers(conn, logger, cachedStore, workerCount, consumerTracer)
	go BookingRedisConsumers(conn, logger, cachedStore, workerCount, consumerTracer)
	go BookingMongoDBConsumers(conn, logger, cachedStore, workerCount, consumerTracer)
	return nil
}
