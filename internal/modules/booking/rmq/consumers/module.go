package consumers

import (
	"context"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	bookingRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq"
)

// StartConsumers starts all consumers for the booking module and returns a
// stop function. stop cancels every consumer and waits for in-flight work to
func StartConsumers(conn *amqp091.Connection, cacheStore *bookingCacheStore.CacheStore, logger *logrus.Logger, traceProvider trace.TracerProvider, rmqCfg configs.RMQConfig) func(stopCtx context.Context) error {
	// Create one tracer for the entire consumer layer
	consumerTracer := traceProvider.Tracer(constants.TracerBookingConsumers)

	// Declare the durable DLQ before any pool starts so dead letters are retained from the first message
	if err := baseRabbitMQ.DeclareDLQ(conn, string(bookingRMQ.ExchangeDLX), string(bookingRMQ.QueueBookingDLQ), logger); err != nil {
		logger.WithError(err).Error("Failed to declare booking DLQ topology")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// MongoDB projection consumer disabled: MongoDB is not used anywhere right
	wg.Add(2)
	go func() {
		defer wg.Done()
		StartPostgresConsumer(ctx, conn, logger, cacheStore, rmqCfg.Consumers.Postgres, consumerTracer)
	}()
	// go func() {
	// 	defer wg.Done()
	// 	StartMongoDBConsumer(ctx, conn, logger, cacheStore, 2, consumerTracer)
	// }()
	go func() {
		defer wg.Done()
		StartPaymentEventsConsumer(ctx, conn, logger, cacheStore, 2, consumerTracer)
	}()

	return func(stopCtx context.Context) error {
		cancel()
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			return stopCtx.Err()
		}
	}
}

// runConsumerPool starts workerCount workers consuming the queue described by cfg and blocks until ctx is cancelled
func runConsumerPool(
	ctx context.Context,
	conn *amqp091.Connection,
	cfg baseRabbitMQ.ConsumerConfig,
	name string,
	workerCount int,
	handler func(context.Context, amqp091.Delivery) error,
	logger *logrus.Logger,
	tracer trace.Tracer,
) {
	var wg sync.WaitGroup

	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			consumer := baseRabbitMQ.NewConsumer(conn, cfg, tracer, handler, newDLQHandler(logger, name), logger)
			logger.WithFields(logrus.Fields{"consumer": name, "worker": id}).Info("consumer worker started")

			if err := consumer.Start(ctx); err != nil {
				logger.WithError(err).WithFields(logrus.Fields{"consumer": name, "worker": id}).Error("consumer worker crashed")
			}
		}(i)
	}

	// Blocks until ctx is cancelled: each worker's Start drains
	wg.Wait()
}
