package bookingConsumers

import (
	"context"
	"fmt"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	bookingRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq"
)

// Creates redis consumers for booking events
func BookingRedisConsumers(conn *amqp091.Connection, logger *logrus.Logger, cacheStore *bookingCacheStore.CacheStore, workerCount int, tracer trace.Tracer) {

	handleRedis := func(ctx context.Context, msg amqp091.Delivery) error {
		// TODO: DB insert logic
		fmt.Println("Redis worker", string(msg.Body))
		return nil
	}

	handleDLQ := func(ctx context.Context, msg amqp091.Delivery) error {
		fmt.Println("DLQ worker", string(msg.Body))
		return nil
	}

	// TODO: make it configurable in yml file
	cfg := baseRabbitMQ.ConsumerConfig{
		ExchangeName:  string(bookingRMQ.ExchangeBooking),
		RoutingKey:    string(bookingRMQ.BindingKeyBookingAll),
		QueueName:     string(bookingRMQ.QueueBookingRedis),
		DLXExchange:   string(bookingRMQ.ExchangeDLX),
		DLXRoutingKey: string(bookingRMQ.BindingKeyBookingAll),
		PrefetchCount: 1, // each worker holds only 1 unacked message
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			consumer := baseRabbitMQ.NewConsumer(conn, cfg, tracer, handleRedis, handleDLQ, logger)
			logger.WithField("worker", id).Infof("Redis worker started")

			if err := consumer.Start(ctx); err != nil {
				logger.WithError(err).WithField("worker", id).Error("Redis worker crashed")
			}
		}(i)
	}

	// Blocks forever until ctx is cancelled (shutdown signal)
	wg.Wait()
}
